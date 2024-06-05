package jmap

import (
	"encoding/json"
	"io"
	"strings"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail"
	"git.sr.ht/~rockorager/go-jmap/mail/identity"
)

func (w *JMAPWorker) handleConnect(msg *types.Connect) error {
	w.client = &jmap.Client{SessionEndpoint: w.config.endpoint}

	if w.config.oauth {
		pass, _ := w.config.user.Password()
		w.client.WithAccessToken(pass)
	} else {
		user := w.config.user.Username()
		pass, _ := w.config.user.Password()
		w.client.WithBasicAuth(user, pass)
	}

	if session, err := w.cache.GetSession(); err == nil {
		w.client.Session = session
		if w.GetIdentities() != nil {
			w.client.Session = nil
			w.identities = make(map[string]*identity.Identity)
			if err := w.cache.DeleteSession(); err != nil {
				w.w.Warnf("DeleteSession: %s", err)
			}
		}
	}
	if w.client.Session == nil {
		if err := w.client.Authenticate(); err != nil {
			return err
		}
		if err := w.cache.PutSession(w.client.Session); err != nil {
			w.w.Warnf("PutSession: %s", err)
		}
	}
	if len(w.identities) == 0 {
		if err := w.GetIdentities(); err != nil {
			return err
		}
	}

	return nil
}

func (w *JMAPWorker) AccountId() jmap.ID {
	switch {
	case w.client == nil:
		fallthrough
	case w.client.Session == nil:
		fallthrough
	case w.client.Session.PrimaryAccounts == nil:
		return ""
	default:
		return w.client.Session.PrimaryAccounts[mail.URI]
	}
}

func (w *JMAPWorker) GetIdentities() error {
	var req jmap.Request

	req.Invoke(&identity.Get{Account: w.AccountId()})
	resp, err := w.Do(&req)
	if err != nil {
		return err
	}
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *identity.GetResponse:
			for _, ident := range r.List {
				w.identities[ident.Email] = ident
			}
		case *jmap.MethodError:
			return wrapMethodError(r)
		}
	}

	return nil
}

var seqnum uint64

func (w *JMAPWorker) Do(req *jmap.Request) (*jmap.Response, error) {
	seq := atomic.AddUint64(&seqnum, 1)
	body, _ := json.Marshal(req.Calls)
	w.w.Debugf(">%d> POST %s", seq, body)
	resp, err := w.client.Do(req)
	if err == nil {
		w.w.Debugf("<%d< done", seq)
	} else {
		w.w.Debugf("<%d< %s", seq, err)
	}
	return resp, err
}

func (w *JMAPWorker) Download(blobID jmap.ID) (io.ReadCloser, error) {
	seq := atomic.AddUint64(&seqnum, 1)
	replacer := strings.NewReplacer(
		"{accountId}", string(w.AccountId()),
		"{blobId}", string(blobID),
		"{type}", "application/octet-stream",
		"{name}", "filename",
	)
	url := replacer.Replace(w.client.Session.DownloadURL)
	w.w.Debugf(">%d> GET %s", seq, url)
	rd, err := w.client.Download(w.AccountId(), blobID)
	if err == nil {
		w.w.Debugf("<%d< 200 OK", seq)
	} else {
		w.w.Debugf("<%d< %s", seq, err)
	}
	return rd, err
}

func (w *JMAPWorker) Upload(reader io.Reader) (*jmap.UploadResponse, error) {
	seq := atomic.AddUint64(&seqnum, 1)
	url := strings.ReplaceAll(w.client.Session.UploadURL,
		"{accountId}", string(w.AccountId()))
	w.w.Debugf(">%d> POST %s", seq, url)
	resp, err := w.client.Upload(w.AccountId(), reader)
	if err == nil {
		w.w.Debugf("<%d< 200 OK", seq)
	} else {
		w.w.Debugf("<%d< %s", seq, err)
	}
	return resp, err
}
