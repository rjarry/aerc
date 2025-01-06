package send

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"

	"github.com/emersion/go-message/mail"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

// NewSender returns an io.WriterCloser into which the caller can write
// contents of a message. The caller must invoke the Close() method on the
// sender when finished.
func NewSender(
	worker *types.Worker, uri *url.URL, domain string,
	from *mail.Address, rcpts []*mail.Address,
	copyTo []string,
) (io.WriteCloser, error) {
	protocol, auth, err := parseScheme(uri)
	if err != nil {
		return nil, err
	}

	var w io.WriteCloser

	switch protocol {
	case "smtp", "smtp+insecure", "smtps":
		w, err = newSmtpSender(protocol, auth, uri, domain, from, rcpts)
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
	defer w.w.Close() // ensure closed even on error

	scan := bufio.NewScanner(&w.buf)
	for scan.Scan() {
		if _, err := w.w.Write(append(scan.Bytes(), '\r', '\n')); err != nil {
			return nil
		}
	}
	if scan.Err() != nil {
		return scan.Err()
	}

	return w.w.Close()
}
