package cryptoutil

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"github.com/emersion/go-message/mail"
)

func Cleartext(r io.Reader, header mail.Header) ([]byte, error) {
	msg, err := app.CryptoProvider().Decrypt(
		rfc822.NewCRLFReader(r), app.DecryptKeys)
	if err != nil {
		return nil, errors.New("decrypt error")
	}
	full, err := createMessage(header, msg.Body)
	if err != nil {
		return nil, errors.New("failed to create decrypted message")
	}
	return full, nil
}

func createMessage(header mail.Header, body io.Reader) ([]byte, error) {
	e, err := rfc822.ReadMessage(body)
	if err != nil {
		return nil, err
	}

	// copy the header values from the "decrypted body". This should set
	// the correct content type.
	hf := e.Header.Fields()
	for hf.Next() {
		header.Set(hf.Key(), hf.Value())
	}

	ctype, params, err := header.ContentType()
	if err != nil {
		return nil, err
	}

	// in case there remains a multipart/{encrypted,signed} content type,
	// manually correct them to multipart/mixed as a fallback.
	ct := strings.ToLower(ctype)
	if strings.Contains(ct, "multipart/encrypted") ||
		strings.Contains(ct, "multipart/signed") {
		delete(params, "protocol")
		delete(params, "micalg")
		header.SetContentType("multipart/mixed", params)
	}

	// a SingleInlineWriter is sufficient since the "decrypted body"
	// already contains the proper boundaries of the parts; we just want to
	// combine it with the headers.
	var message bytes.Buffer
	w, err := mail.CreateSingleInlineWriter(&message, header)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(w, e.Body); err != nil {
		return nil, err
	}
	w.Close()

	return message.Bytes(), nil
}
