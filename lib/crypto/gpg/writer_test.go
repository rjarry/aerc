package gpg

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/crypto/gpg/gpgbin"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/textproto"
)

func init() {
	forceBoundary = "foo"
}

func TestEncrypt(t *testing.T) {
	importPublicKey()
	importSecretKey()
	var h textproto.Header
	h.Set("From", "John Doe <john.doe@example.org>")
	h.Set("To", "John Doe <john.doe@example.org>")

	var encryptedHeader textproto.Header
	encryptedHeader.Set("Content-Type", "text/plain")

	var encryptedBody = "This is an encrypted message!\r\n"

	to := []string{"john.doe@example.org"}
	from := "john.doe@example.org"

	var buf bytes.Buffer
	cleartext, err := Encrypt(&buf, h, to, from)
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}

	if err = textproto.WriteHeader(cleartext, encryptedHeader); err != nil {
		t.Fatalf("textproto.WriteHeader() = %v", err)
	}
	if _, err = io.WriteString(cleartext, encryptedBody); err != nil {
		t.Fatalf("io.WriteString() = %v", err)
	}
	if err = cleartext.Close(); err != nil {
		t.Fatalf("ciphertext.Close() = %v", err)
	}

	md, err := gpgbin.Decrypt(&buf)
	if err != nil {
		t.Errorf("Encrypt error: could not decrypt test encryption")
	}
	var body bytes.Buffer
	io.Copy(&body, md.Body)
	if s := body.String(); s != wantEncrypted {
		t.Errorf("Encrypt() = \n%v\n but want \n%v", s, wantEncrypted)
	}

	t.Cleanup(CleanUp)
}

func TestSign(t *testing.T) {
	importPublicKey()
	importSecretKey()
	var h textproto.Header
	h.Set("From", "John Doe <john.doe@example.org>")
	h.Set("To", "John Doe <john.doe@example.org>")

	var signedHeader textproto.Header
	signedHeader.Set("Content-Type", "text/plain")

	var signedBody = "This is a signed message!\r\n"

	var buf bytes.Buffer
	cleartext, err := Sign(&buf, h, "john.doe@example.org")
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}

	if err = textproto.WriteHeader(cleartext, signedHeader); err != nil {
		t.Fatalf("textproto.WriteHeader() = %v", err)
	}
	if _, err = io.WriteString(cleartext, signedBody); err != nil {
		t.Fatalf("io.WriteString() = %v", err)
	}

	if err = cleartext.Close(); err != nil {
		t.Fatalf("ciphertext.Close() = %v", err)
	}

	parts := strings.Split(buf.String(), "\r\n--foo\r\n")
	msg := strings.NewReader(parts[1])
	sig := strings.NewReader(parts[2])
	md, err := gpgbin.Verify(msg, sig)
	if err != nil {
		t.Fatalf("gpg.Verify() = %v", err)
	}

	deepEqual(t, md, &wantSigned)
}

var wantEncrypted = toCRLF(`Content-Type: text/plain

This is an encrypted message!
`)

var wantSignedBody = toCRLF(`Content-Type: text/plain

This is a signed message!
`)

var wantSigned = models.MessageDetails{
	IsEncrypted:        false,
	IsSigned:           true,
	SignedBy:           "John Doe (This is a test key) <john.doe@example.org>",
	SignedByKeyId:      3490876580878068068,
	SignatureError:     "",
	DecryptedWith:      "",
	DecryptedWithKeyId: 0,
	Body:               strings.NewReader(wantSignedBody),
	Micalg:             "pgp-sha256",
}
