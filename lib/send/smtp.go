package send

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	"github.com/pkg/errors"
)

func connectSmtp(starttls bool, host string, domain string) (*smtp.Client, error) {
	serverName := host
	if !strings.ContainsRune(host, ':') {
		host += ":587" // Default to submission port
	} else {
		serverName = host[:strings.IndexRune(host, ':')]
	}
	var conn *smtp.Client
	var err error
	if starttls {
		conn, err = smtp.DialStartTLS(host, &tls.Config{ServerName: serverName})
	} else {
		conn, err = smtp.Dial(host)
	}
	if err != nil {
		return nil, errors.Wrap(err, "smtp.Dial")
	}
	if domain != "" {
		err := conn.Hello(domain)
		if err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "Hello")
		}
	}
	return conn, nil
}

func connectSmtps(host string, domain string) (*smtp.Client, error) {
	serverName := host
	if !strings.ContainsRune(host, ':') {
		host += ":465" // Default to smtps port
	} else {
		serverName = host[:strings.IndexRune(host, ':')]
	}
	conn, err := smtp.DialTLS(host, &tls.Config{
		ServerName: serverName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "smtp.DialTLS")
	}
	if domain != "" {
		err := conn.Hello(domain)
		if err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "Hello")
		}
	}
	return conn, nil
}

type smtpSender struct {
	conn *smtp.Client
	w    io.WriteCloser
}

func (s *smtpSender) Write(p []byte) (int, error) {
	return s.w.Write(p)
}

func (s *smtpSender) Close() error {
	we := s.w.Close()
	ce := s.conn.Close()
	if we != nil {
		return we
	}
	return ce
}

func newSmtpSender(
	protocol string, auth string, uri *url.URL, domain string,
	from *mail.Address, rcpts []*mail.Address,
) (io.WriteCloser, error) {
	var err error
	var conn *smtp.Client
	switch protocol {
	case "smtp":
		conn, err = connectSmtp(true, uri.Host, domain)
	case "smtp+insecure":
		conn, err = connectSmtp(false, uri.Host, domain)
	case "smtps":
		conn, err = connectSmtps(uri.Host, domain)
	default:
		return nil, fmt.Errorf("not a smtp protocol %s", protocol)
	}

	if err != nil {
		return nil, errors.Wrap(err, "Connection failed")
	}

	saslclient, err := newSaslClient(auth, uri)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if saslclient != nil {
		if err := conn.Auth(saslclient); err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "conn.Auth")
		}
	}
	s := &smtpSender{
		conn: conn,
	}
	if err := s.conn.Mail(from.Address, nil); err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "conn.Mail")
	}
	for _, rcpt := range rcpts {
		if err := s.conn.Rcpt(rcpt.Address, nil); err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "conn.Rcpt")
		}
	}
	s.w, err = s.conn.Data()
	if err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "conn.Data")
	}
	return s.w, nil
}
