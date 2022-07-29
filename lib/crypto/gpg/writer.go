// writer.go largerly mimics github.com/emersion/go-pgpmail, with changes made
// to interface with the gpg package in aerc

package gpg

import (
	"bytes"
	"fmt"
	"io"
	"mime"

	"git.sr.ht/~rjarry/aerc/lib/crypto/gpg/gpgbin"
	"github.com/emersion/go-message/textproto"
)

type EncrypterSigner struct {
	msgBuf          bytes.Buffer
	encryptedWriter io.Writer
	to              []string
	from            string
}

func (es *EncrypterSigner) Write(p []byte) (int, error) {
	return es.msgBuf.Write(p)
}

func (es *EncrypterSigner) Close() (err error) {
	r := bytes.NewReader(es.msgBuf.Bytes())
	enc, err := gpgbin.Encrypt(r, es.to, es.from)
	if err != nil {
		return err
	}
	_, err = es.encryptedWriter.Write(enc)
	if err != nil {
		return fmt.Errorf("gpg: failed to write encrypted writer: %w", err)
	}
	return nil
}

type Signer struct {
	mw        *textproto.MultipartWriter
	signedMsg bytes.Buffer
	w         io.Writer
	from      string
	header    textproto.Header
}

func (s *Signer) Write(p []byte) (int, error) {
	return s.signedMsg.Write(p)
}

func (s *Signer) Close() (err error) {
	// TODO should write the whole message up here so we can get the proper micalg from the signature packet

	sig, micalg, err := gpgbin.Sign(bytes.NewReader(s.signedMsg.Bytes()), s.from)
	if err != nil {
		return err
	}
	params := map[string]string{
		"boundary": s.mw.Boundary(),
		"protocol": "application/pgp-signature",
		"micalg":   micalg,
	}
	s.header.Set("Content-Type", mime.FormatMediaType("multipart/signed", params))

	if err = textproto.WriteHeader(s.w, s.header); err != nil {
		return err
	}
	boundary := s.mw.Boundary()
	fmt.Fprintf(s.w, "--%s\r\n", boundary)
	_, _ = s.w.Write(s.signedMsg.Bytes())
	_, _ = s.w.Write([]byte("\r\n"))

	var signedHeader textproto.Header
	signedHeader.Set("Content-Type", "application/pgp-signature; name=\"signature.asc\"")
	signatureWriter, err := s.mw.CreatePart(signedHeader)
	if err != nil {
		return err
	}

	_, err = signatureWriter.Write(sig)
	if err != nil {
		return err
	}
	return nil
}

// for tests
var forceBoundary = ""

type multiCloser []io.Closer

func (mc multiCloser) Close() error {
	for _, c := range mc {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
}

func Encrypt(w io.Writer, h textproto.Header, rcpts []string, from string) (io.WriteCloser, error) {
	mw := textproto.NewMultipartWriter(w)

	if forceBoundary != "" {
		err := mw.SetBoundary(forceBoundary)
		if err != nil {
			return nil, fmt.Errorf("gpg: failed to set boundary: %w", err)
		}
	}

	params := map[string]string{
		"boundary": mw.Boundary(),
		"protocol": "application/pgp-encrypted",
	}
	h.Set("Content-Type", mime.FormatMediaType("multipart/encrypted", params))

	if err := textproto.WriteHeader(w, h); err != nil {
		return nil, err
	}

	var controlHeader textproto.Header
	controlHeader.Set("Content-Type", "application/pgp-encrypted")
	controlWriter, err := mw.CreatePart(controlHeader)
	if err != nil {
		return nil, err
	}
	if _, err = controlWriter.Write([]byte("Version: 1\r\n")); err != nil {
		return nil, err
	}

	var encryptedHeader textproto.Header
	encryptedHeader.Set("Content-Type", "application/octet-stream")
	encryptedWriter, err := mw.CreatePart(encryptedHeader)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	plaintext := &EncrypterSigner{
		msgBuf:          buf,
		encryptedWriter: encryptedWriter,
		to:              rcpts,
		from:            from,
	}

	return struct {
		io.Writer
		io.Closer
	}{
		plaintext,
		multiCloser{
			plaintext,
			mw,
		},
	}, nil
}

func Sign(w io.Writer, h textproto.Header, from string) (io.WriteCloser, error) {
	mw := textproto.NewMultipartWriter(w)

	if forceBoundary != "" {
		err := mw.SetBoundary(forceBoundary)
		if err != nil {
			return nil, fmt.Errorf("gpg: failed to set boundary: %w", err)
		}
	}

	var msg bytes.Buffer
	plaintext := &Signer{
		mw:        mw,
		signedMsg: msg,
		w:         w,
		from:      from,
		header:    h,
	}

	return struct {
		io.Writer
		io.Closer
	}{
		plaintext,
		multiCloser{
			plaintext,
			mw,
		},
	}, nil
}
