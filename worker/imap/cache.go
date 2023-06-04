package imap

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"
	"github.com/mitchellh/go-homedir"
	"github.com/syndtr/goleveldb/leveldb"
)

type CachedHeader struct {
	BodyStructure models.BodyStructure
	Envelope      models.Envelope
	InternalDate  time.Time
	Uid           uint32
	Size          uint32
	Header        []byte
	Created       time.Time
}

var (
	// cacheTag should be updated when changing the cache
	// structure; this will ensure that the user's cache is cleared and
	// reloaded when the underlying cache structure changes
	cacheTag    = []byte("0001")
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
	cd, err := cacheDir()
	if err != nil {
		w.cache = nil
		w.worker.Errorf("unable to find cache directory: %v", err)
		return
	}
	p := path.Join(cd, acct)
	db, err := leveldb.OpenFile(p, nil)
	if err != nil {
		w.cache = nil
		w.worker.Errorf("failed opening cache db: %v", err)
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
		w.worker.Tracef("cache version match")
		if w.config.cacheMaxAge.Hours() > 0 {
			go w.cleanCache(p)
		}
	}
}

func (w *IMAPWorker) cacheHeader(mi *models.MessageInfo) {
	uv := fmt.Sprintf("%d", w.selected.UidValidity)
	uid := fmt.Sprintf("%d", mi.Uid)
	w.worker.Debugf("caching header for message %s.%s", uv, uid)
	hdr := bytes.NewBuffer(nil)
	err := textproto.WriteHeader(hdr, mi.RFC822Headers.Header.Header)
	if err != nil {
		w.worker.Errorf("cannot write header %s.%s: %v", uv, uid, err)
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
	}
	data := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(data)
	err = enc.Encode(h)
	if err != nil {
		w.worker.Errorf("cannot encode message %s.%s: %v", uv, uid, err)
		return
	}
	err = w.cache.Put([]byte("header."+uv+"."+uid), data.Bytes(), nil)
	if err != nil {
		w.worker.Errorf("cannot write header for message %s.%s: %v", uv, uid, err)
		return
	}
}

func (w *IMAPWorker) getCachedHeaders(msg *types.FetchMessageHeaders) []uint32 {
	w.worker.Tracef("Retrieving headers from cache: %v", msg.Uids)
	var need []uint32
	uv := fmt.Sprintf("%d", w.selected.UidValidity)
	for _, uid := range msg.Uids {
		u := fmt.Sprintf("%d", uid)
		data, err := w.cache.Get([]byte("header."+uv+"."+u), nil)
		if err != nil {
			need = append(need, uid)
			continue
		}
		ch := &CachedHeader{}
		dec := gob.NewDecoder(bytes.NewReader(data))
		err = dec.Decode(ch)
		if err != nil {
			w.worker.Errorf("cannot decode cached header %s.%s: %v", uv, u, err)
			need = append(need, uid)
			continue
		}
		hr := bytes.NewReader(ch.Header)
		textprotoHeader, err := textproto.ReadHeader(bufio.NewReader(hr))
		if err != nil {
			w.worker.Errorf("cannot read cached header %s.%s: %v", uv, u, err)
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
		}
		w.worker.Tracef("located cached header %s.%s", uv, u)
		w.worker.PostMessage(&types.MessageInfo{
			Message:    types.RespondTo(msg),
			Info:       mi,
			NeedsFlags: true,
		}, nil)
	}
	return need
}

func cacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir, err = homedir.Expand("~/.cache")
		if err != nil {
			return "", err
		}
	}
	return path.Join(dir, "aerc"), nil
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
