package jmap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail"
	"git.sr.ht/~rockorager/go-jmap/mail/identity"
)

func (w *JMAPWorker) handleConnect(msg *types.Connect) error {
	client := &jmap.Client{SessionEndpoint: w.config.endpoint}

	if w.config.oauth {
		pass, _ := w.config.user.Password()
		client.WithAccessToken(pass)
	} else {
		user := w.config.user.Username()
		pass, _ := w.config.user.Password()
		client.WithBasicAuth(user, pass)
	}

	if session, err := w.cache.GetSession(); err != nil {
		if err := client.Authenticate(); err != nil {
			return err
		}
		if err := w.cache.PutSession(client.Session); err != nil {
			w.w.Warnf("PutSession: %s", err)
		}
	} else {
		client.Session = session
	}

	switch {
	case client == nil:
		fallthrough
	case client.Session == nil:
		fallthrough
	case client.Session.PrimaryAccounts == nil:
		break
	default:
		w.accountId = client.Session.PrimaryAccounts[mail.URI]
	}

	w.client = client

	return w.GetIdentities()
}

func (w *JMAPWorker) GetIdentities() error {
	u, err := url.Parse(w.config.account.Outgoing.Value)
	if err != nil {
		return fmt.Errorf("GetIdentities: %w", err)
	}
	if !strings.HasPrefix(u.Scheme, "jmap") {
		// no need for identities
		return nil
	}

	var req jmap.Request

	req.Invoke(&identity.Get{Account: w.accountId})
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
		"{accountId}", string(w.accountId),
		"{blobId}", string(blobID),
		"{type}", "application/octet-stream",
		"{name}", "filename",
	)
	url := replacer.Replace(w.client.Session.DownloadURL)
	w.w.Debugf(">%d> GET %s", seq, url)
	rd, err := w.client.Download(w.accountId, blobID)
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
		"{accountId}", string(w.accountId))
	w.w.Debugf(">%d> POST %s", seq, url)
	resp, err := w.client.Upload(w.accountId, reader)
	if err == nil {
		w.w.Debugf("<%d< 200 OK", seq)
	} else {
		w.w.Debugf("<%d< %s", seq, err)
	}
	return resp, err
}
