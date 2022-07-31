// reader.go largerly mimics github.com/emersion/go-gpgmail, with changes made
// to interface with the gpg package in aerc

package gpg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/crypto/gpg/gpgbin"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/textproto"
)

type Reader struct {
	Header         textproto.Header
	MessageDetails *models.MessageDetails
}

func NewReader(h textproto.Header, body io.Reader) (*Reader, error) {
	t, params, err := mime.ParseMediaType(h.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(t, "multipart/encrypted") && strings.EqualFold(params["protocol"], "application/pgp-encrypted") {
		mr := textproto.NewMultipartReader(body, params["boundary"])
		return newEncryptedReader(h, mr)
	}
	if strings.EqualFold(t, "multipart/signed") && strings.EqualFold(params["protocol"], "application/pgp-signature") {
		micalg := params["micalg"]
		mr := textproto.NewMultipartReader(body, params["boundary"])
		return newSignedReader(h, mr, micalg)
	}

	var headerBuf bytes.Buffer
	_ = textproto.WriteHeader(&headerBuf, h)

	return &Reader{
		Header: h,
		MessageDetails: &models.MessageDetails{
			Body: io.MultiReader(&headerBuf, body),
		},
	}, nil
}

func Read(r io.Reader) (*Reader, error) {
	br := bufio.NewReader(r)

	h, err := textproto.ReadHeader(br)
	if err != nil {
		return nil, err
	}
	return NewReader(h, br)
}

func newEncryptedReader(h textproto.Header, mr *textproto.MultipartReader) (*Reader, error) {
	p, err := mr.NextPart()
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to read first part in multipart/encrypted message: %w", err)
	}

	t, _, err := mime.ParseMediaType(p.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to parse Content-Type of first part in multipart/encrypted message: %w", err)
	}
	if !strings.EqualFold(t, "application/pgp-encrypted") {
		return nil, fmt.Errorf("gpgmail: first part in multipart/encrypted message has type %q, not application/pgp-encrypted", t)
	}

	metadata, err := textproto.ReadHeader(bufio.NewReader(p))
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to parse application/pgp-encrypted part: %w", err)
	}
	if s := metadata.Get("Version"); s != "1" {
		return nil, fmt.Errorf("gpgmail: unsupported PGP/MIME version: %q", s)
	}

	p, err = mr.NextPart()
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to read second part in multipart/encrypted message: %w", err)
	}
	t, _, err = mime.ParseMediaType(p.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to parse Content-Type of second part in multipart/encrypted message: %w", err)
	}
	if !strings.EqualFold(t, "application/octet-stream") {
		return nil, fmt.Errorf("gpgmail: second part in multipart/encrypted message has type %q, not application/octet-stream", t)
	}

	md, err := gpgbin.Decrypt(p)
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to read PGP message: %w", err)
	}

	cleartext := bufio.NewReader(md.Body)
	cleartextHeader, err := textproto.ReadHeader(cleartext)
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to read encrypted header: %w", err)
	}

	t, params, err := mime.ParseMediaType(cleartextHeader.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	if md.IsEncrypted && !md.IsSigned && strings.EqualFold(t, "multipart/signed") && strings.EqualFold(params["protocol"], "application/pgp-signature") {
		// RFC 1847 encapsulation, see RFC 3156 section 6.1
		micalg := params["micalg"]
		mr := textproto.NewMultipartReader(cleartext, params["boundary"])
		mds, err := newSignedReader(cleartextHeader, mr, micalg)
		if err != nil {
			return nil, fmt.Errorf("gpgmail: failed to read encapsulated multipart/signed message: %w", err)
		}
		mds.MessageDetails.IsEncrypted = md.IsEncrypted
		mds.MessageDetails.DecryptedWith = md.DecryptedWith
		mds.MessageDetails.DecryptedWithKeyId = md.DecryptedWithKeyId
		return mds, nil
	}

	var headerBuf bytes.Buffer
	_ = textproto.WriteHeader(&headerBuf, cleartextHeader)
	md.Body = io.MultiReader(&headerBuf, cleartext)

	return &Reader{
		Header:         h,
		MessageDetails: md,
	}, nil
}

func newSignedReader(h textproto.Header, mr *textproto.MultipartReader, micalg string) (*Reader, error) {
	micalg = strings.ToLower(micalg)
	p, err := mr.NextPart()
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to read signed part in multipart/signed message: %w", err)
	}
	var headerBuf bytes.Buffer
	_ = textproto.WriteHeader(&headerBuf, p.Header)
	var msg bytes.Buffer
	headerRdr := bytes.NewReader(headerBuf.Bytes())
	fullMsg := io.MultiReader(headerRdr, p)
	_, _ = io.Copy(&msg, fullMsg)

	sig, err := mr.NextPart()
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to read pgp part in multipart/signed message: %w", err)
	}

	md, err := gpgbin.Verify(&msg, sig)
	if err != nil {
		return nil, fmt.Errorf("gpgmail: failed to read PGP message: %w", err)
	}
	if md.Micalg != micalg && md.SignatureError == "" {
		md.SignatureError = "gpg: header hash does not match actual sig hash"
	}

	return &Reader{
		Header:         h,
		MessageDetails: md,
	}, nil
}
