package send

import (
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

	switch protocol {
	case "smtp", "smtp+insecure", "smtps":
		return newSmtpSender(protocol, auth, uri, domain, from, rcpts)
	case "jmap":
		return newJmapSender(worker, from, rcpts, copyTo)
	case "":
		return newSendmailSender(uri, rcpts)
	default:
		return nil, fmt.Errorf("unsupported protocol %s", protocol)
	}
}
