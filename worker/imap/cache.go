package imap

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"
	"github.com/syndtr/goleveldb/leveldb"
)

type CachedHeader struct {
	BodyStructure models.BodyStructure
	Envelope      models.Envelope
	InternalDate  time.Time
	Uid           models.UID
	Size          uint32
	Header        []byte
	Created       time.Time
	Labels        []string
}

var (
	// cacheTag should be updated when changing the cache
	// structure; this will ensure that the user's cache is cleared and
	// reloaded when the underlying cache structure changes
	cacheTag    = []byte("0003")
	cacheTagKey = []byte("cache.tag")
)

// initCacheDb opens (or creates) the database for the cache. One database is
// created per account
func (w *IMAPWorker) initCacheDb(acct string) {
	switch {
	case len(w.config.headersExclude) > 0:
		headerTag := strings.Join(w.config.headersExclude, "")
		cacheTag = append(cacheTag, headerTag...)
	case len(w.config.headers) > 0:
		headerTag := strings.Join(w.config.headers, "")
		cacheTag = append(cacheTag, headerTag...)
	}
	p := xdg.CachePath("aerc", acct)
	db, err := leveldb.OpenFile(p, nil)
	if err != nil {
		w.cache = nil
		w.worker.Errorf("failed opening cache db at %s: %v", p, err)
		return
	}
	w.cache = db
	w.worker.Debugf("cache db opened: %s", p)

	tag, err := w.cache.Get(cacheTagKey, nil)
	clearCache := errors.Is(err, leveldb.ErrNotFound) ||
		!bytes.Equal(tag, cacheTag)
	switch {
	case clearCache:
		w.worker.Infof("current cache tag is '%s' but found '%s'",
			cacheTag, tag)
		w.worker.Warnf("tag mismatch: clear cache")
		w.clearCache()
		if err = w.cache.Put(cacheTagKey, cacheTag, nil); err != nil {
			w.worker.Errorf("could not set the current cache tag")
		}
	case err != nil:
		w.worker.Errorf("could not get the cache tag from db")
	default:
		if w.config.cacheMaxAge.Hours() > 0 {
			go w.cleanCache(p)
		}
	}
}

func (w *IMAPWorker) cacheHeader(mi *models.MessageInfo) {
	key := w.headerKey(mi.Uid)
	w.worker.Debugf("caching header for message %s", key)
	hdr := bytes.NewBuffer(nil)
	err := textproto.WriteHeader(hdr, mi.RFC822Headers.Header.Header)
	if err != nil {
		w.worker.Errorf("cannot write header %s: %v", key, err)
		return
	}
	h := &CachedHeader{
		BodyStructure: *mi.BodyStructure,
		Envelope:      *mi.Envelope,
		InternalDate:  mi.InternalDate,
		Uid:           mi.Uid,
		Size:          mi.Size,
		Header:        hdr.Bytes(),
		Created:       time.Now(),
		Labels:        mi.Labels,
	}
	data := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(data)
	err = enc.Encode(h)
	if err != nil {
		w.worker.Errorf("cannot encode message %s: %v", key, err)
		return
	}
	err = w.cache.Put(key, data.Bytes(), nil)
	if err != nil {
		w.worker.Errorf("cannot write header for message %s: %v", key, err)
		return
	}
}

func (w *IMAPWorker) getCachedHeaders(msg *types.FetchMessageHeaders) []models.UID {
	w.worker.Tracef("Retrieving headers from cache: %v", msg.Uids)
	var need []models.UID
	for _, uid := range msg.Uids {
		key := w.headerKey(uid)
		data, err := w.cache.Get(key, nil)
		if err != nil {
			need = append(need, uid)
			continue
		}
		ch := &CachedHeader{}
		dec := gob.NewDecoder(bytes.NewReader(data))
		err = dec.Decode(ch)
		if err != nil {
			w.worker.Errorf("cannot decode cached header %s: %v", key, err)
			need = append(need, uid)
			continue
		}
		hr := bytes.NewReader(ch.Header)
		textprotoHeader, err := textproto.ReadHeader(bufio.NewReader(hr))
		if err != nil {
			w.worker.Errorf("cannot read cached header %s: %v", key, err)
			need = append(need, uid)
			continue
		}

		hdr := &mail.Header{Header: message.Header{Header: textprotoHeader}}
		mi := &models.MessageInfo{
			BodyStructure: &ch.BodyStructure,
			Envelope:      &ch.Envelope,
			Flags:         models.SeenFlag, // Always return a SEEN flag
			Uid:           ch.Uid,
			RFC822Headers: hdr,
			Refs:          parse.MsgIDList(hdr, "references"),
			Size:          ch.Size,
			Labels:        ch.Labels,
		}
		w.worker.PostMessage(&types.MessageInfo{
			Message:    types.RespondTo(msg),
			Info:       mi,
			NeedsFlags: true,
		}, nil)
	}
	return need
}

func (w *IMAPWorker) headerKey(uid models.UID) []byte {
	key := fmt.Sprintf("header.%s.%d.%s",
		w.selected.Name, w.selected.UidValidity, uid)
	return []byte(key)
}

// cleanCache removes stale entries from the selected mailbox cachedb
func (w *IMAPWorker) cleanCache(path string) {
	defer log.PanicHandler()
	start := time.Now()
	var scanned, removed int
	iter := w.cache.NewIterator(nil, nil)
	for iter.Next() {
		if bytes.Equal(iter.Key(), cacheTagKey) {
			continue
		}
		data := iter.Value()
		ch := &CachedHeader{}
		dec := gob.NewDecoder(bytes.NewReader(data))
		err := dec.Decode(ch)
		if err != nil {
			w.worker.Errorf("cannot clean database %d: %v",
				w.selected.UidValidity, err)
			continue
		}
		exp := ch.Created.Add(w.config.cacheMaxAge)
		if exp.Before(time.Now()) {
			err = w.cache.Delete(iter.Key(), nil)
			if err != nil {
				w.worker.Errorf("cannot clean database %d: %v",
					w.selected.UidValidity, err)
				continue
			}
			removed++
		}
		scanned++
	}
	iter.Release()
	elapsed := time.Since(start)
	w.worker.Debugf("%s: removed %d/%d expired entries in %s",
		path, removed, scanned, elapsed)
}

// clearCache clears the entire cache
func (w *IMAPWorker) clearCache() {
	iter := w.cache.NewIterator(nil, nil)
	for iter.Next() {
		if err := w.cache.Delete(iter.Key(), nil); err != nil {
			w.worker.Errorf("error clearing cache: %v", err)
		}
	}
	iter.Release()
}
