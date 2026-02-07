package jmap

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail"
	"git.sr.ht/~rockorager/go-jmap/mail/identity"
	"golang.org/x/oauth2"
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

	if transport, ok := w.client.HttpClient.Transport.(*oauth2.Transport); ok {
		if httpTransport, ok := transport.Base.(*http.Transport); ok {
			// Enable TCP keepalive to detect dead connections faster
			httpTransport.DialContext = (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 15 * time.Second,
			}).DialContext
		}
	}

	if session, err := w.cache.GetSession(); err == nil {
		w.client.Session = session
	}
	if w.client.Session == nil {
		if err := w.UpdateSession(); err != nil {
			return err
		}
	}

	go w.monitorChanges()

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

func (w *JMAPWorker) UpdateSession() error {
	if err := w.client.Authenticate(); err != nil {
		return err
	}
	if err := w.cache.PutSession(w.client.Session); err != nil {
		w.w.Warnf("PutSession: %s", err)
	}
	return nil
}

func (w *JMAPWorker) GetIdentities() error {
	var req jmap.Request

	req.Invoke(&identity.Get{Account: w.AccountId()})
	resp, err := w.Do(context.TODO(), &req)
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

func (w *JMAPWorker) Do(ctx context.Context, req *jmap.Request) (*jmap.Response, error) {
	if ctx != nil {
		req.Context = ctx
	}
	seq := atomic.AddUint64(&seqnum, 1)
	body, _ := json.Marshal(req.Calls)
	w.w.Debugf(">%d> POST %s", seq, body)
	resp, err := w.client.Do(req)
	if err != nil {
		w.w.Debugf("<%d< %s", seq, err)
		// Try to update session in case an endpoint changed
		err := w.UpdateSession()
		if err != nil {
			return nil, err
		}
		// And try again if we succeeded
		resp, err = w.client.Do(req)
		if err != nil {
			return nil, err
		}
	}
	if resp.SessionState != w.client.Session.State {
		if err := w.UpdateSession(); err != nil {
			return nil, err
		}
	}
	w.w.Debugf("<%d< done", seq)
	return resp, err
}

func (w *JMAPWorker) Download(ctx context.Context, blobID jmap.ID) (io.ReadCloser, error) {
	seq := atomic.AddUint64(&seqnum, 1)
	replacer := strings.NewReplacer(
		"{accountId}", string(w.AccountId()),
		"{blobId}", string(blobID),
		"{type}", "application/octet-stream",
		"{name}", "filename",
	)
	url := replacer.Replace(w.client.Session.DownloadURL)
	w.w.Debugf(">%d> GET %s", seq, url)
	rd, err := w.client.DownloadWithContext(ctx, w.AccountId(), blobID)
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
