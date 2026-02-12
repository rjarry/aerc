package send

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"

	"github.com/emersion/go-message/mail"

	"git.sr.ht/~rjarry/aerc/lib/auth"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// NewSender returns an io.WriterCloser into which the caller can write
// contents of a message. The caller must invoke the Close() method on the
// sender when finished.
func NewSender(
	worker *types.Worker, uri *url.URL, domain string,
	from *mail.Address, rcpts []*mail.Address, account string,
	copyTo []string, requestDSN bool,
) (io.WriteCloser, error) {
	protocol, mech, err := auth.ParseScheme(uri)
	if err != nil {
		return nil, err
	}

	var w io.WriteCloser

	switch protocol {
	case "smtp", "smtp+insecure", "smtps":
		w, err = newSmtpSender(protocol, mech, uri, domain, from, rcpts, account, requestDSN)
	case "jmap":
		w, err = newJmapSender(worker, from, rcpts, copyTo)
	case "":
		w, err = newSendmailSender(uri, rcpts)
	default:
		err = fmt.Errorf("unsupported protocol %s", protocol)
	}
	if err != nil {
		return nil, err
	}
	return &crlfWriter{w: w}, nil
}

type crlfWriter struct {
	w   io.WriteCloser
	buf bytes.Buffer
}

func (w *crlfWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *crlfWriter) Close() error {
	scan := bufio.NewScanner(&w.buf)
	for scan.Scan() {
		if _, err := w.w.Write(append(scan.Bytes(), '\r', '\n')); err != nil {
			w.w.Close()
			return err
		}
	}
	if scan.Err() != nil {
		w.w.Close()
		return scan.Err()
	}
	return w.w.Close()
}
