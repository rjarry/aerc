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

type writerTestCase struct {
	name        string
	method      string
	body        string
	to          []string
	expectedErr string
}

func TestWriter(t *testing.T) {
	initGPGtest(t)
	importSecretKey()
	importPublicKey()
	importOwnertrust()

	testCases := []writerTestCase{
		{
			name:   "Encrypt",
			method: "encrypt",
			body:   "This is an encrypted message!\r\n",
			to:     []string{"john.doe@example.org"},
		},
		{
			name:   "Sign",
			method: "sign",
			body:   "This is a signed message!\r\n",
			to:     []string{"john.doe@example.org"},
		},
		{
			name:        "Encrypt to untrusted",
			method:      "encrypt",
			body:        "This is an encrypted message!\r\n",
			to:          []string{"jane.doe@example.org"},
			expectedErr: "gpg: failure to encrypt: gpg: public key of jane.doe@example.org is not trusted. check public key(s)",
		},
	}
	var h textproto.Header
	h.Set("From", "John Doe <john.doe@example.org>")
	h.Set("To", "John Doe <john.doe@example.org>")

	var header textproto.Header
	header.Set("Content-Type", "text/plain")

	from := "john.doe@example.org"

	var err error
	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		var (
			buf       bytes.Buffer
			cleartext io.WriteCloser
		)
		switch tc.method {
		case "encrypt":
			cleartext, err = Encrypt(&buf, h, tc.to, from)
			if err != nil {
				t.Fatalf("Encrypt() = %v", err)
			}
		case "sign":
			cleartext, err = Sign(&buf, h, from)
			if err != nil {
				t.Fatalf("Encrypt() = %v", err)
			}
		}
		if err = textproto.WriteHeader(cleartext, header); err != nil {
			t.Fatalf("textproto.WriteHeader() = %v", err)
		}
		if _, err = io.WriteString(cleartext, tc.body); err != nil {
			t.Fatalf("io.WriteString() = %v", err)
		}
		if err = cleartext.Close(); err != nil {
			if err.Error() == tc.expectedErr {
				continue
			}
			t.Fatalf("ciphertext.Close() = %v", err)
		}
		if tc.expectedErr != "" {
			t.Fatalf("Expected error %v, but got %v", tc.expectedErr, err)
		}
		switch tc.method {
		case "encrypt":
			validateEncrypt(t, buf)
		case "sign":
			validateSign(t, buf)
		}
	}
}

func validateEncrypt(t *testing.T, buf bytes.Buffer) {
	md, err := gpgbin.Decrypt(&buf)
	if err != nil {
		t.Errorf("Encrypt error: could not decrypt test encryption")
	}
	var body bytes.Buffer
	io.Copy(&body, md.Body)
	if s := body.String(); s != wantEncrypted {
		t.Errorf("Encrypt() = \n%v\n but want \n%v", s, wantEncrypted)
	}
}

func validateSign(t *testing.T, buf bytes.Buffer) {
	parts := strings.Split(buf.String(), "\r\n--foo\r\n")
	msg := strings.NewReader(parts[1])
	sig := strings.NewReader(parts[2])
	md, err := gpgbin.Verify(msg, sig)
	if err != nil {
		t.Fatalf("gpg.Verify() = %v", err)
	}

	deepEqual(t, "Sign", md, &wantSigned)
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
