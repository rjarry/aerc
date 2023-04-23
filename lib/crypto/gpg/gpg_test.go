package gpg

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/models"
)

func initGPGtest(t *testing.T) {
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skipf("%s", err)
	}
	// temp dir is automatically deleted by the test runtime
	dir := t.TempDir()
	t.Setenv("GNUPGHOME", dir)
	t.Logf("using GNUPGHOME = %s", dir)
}

func toCRLF(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func deepEqual(t *testing.T, name string, r *models.MessageDetails, expect *models.MessageDetails) {
	var resBuf bytes.Buffer
	if _, err := io.Copy(&resBuf, r.Body); err != nil {
		t.Fatalf("%s: io.Copy() = %v", name, err)
	}

	var expBuf bytes.Buffer
	if _, err := io.Copy(&expBuf, expect.Body); err != nil {
		t.Fatalf("%s: io.Copy() = %v", name, err)
	}

	if resBuf.String() != expBuf.String() {
		t.Errorf("%s: MessagesDetails.Body = \n%v\n but want \n%v", name, resBuf.String(), expBuf.String())
	}

	if r.IsEncrypted != expect.IsEncrypted {
		t.Errorf("%s: IsEncrypted = \n%v\n but want \n%v", name, r.IsEncrypted, expect.IsEncrypted)
	}
	if r.IsSigned != expect.IsSigned {
		t.Errorf("%s: IsSigned = \n%v\n but want \n%v", name, r.IsSigned, expect.IsSigned)
	}
	if r.SignedBy != expect.SignedBy {
		t.Errorf("%s: SignedBy = \n%v\n but want \n%v", name, r.SignedBy, expect.SignedBy)
	}
	if r.SignedByKeyId != expect.SignedByKeyId {
		t.Errorf("%s: SignedByKeyId = \n%v\n but want \n%v", name, r.SignedByKeyId, expect.SignedByKeyId)
	}
	if r.SignatureError != expect.SignatureError {
		t.Errorf("%s: SignatureError = \n%v\n but want \n%v", name, r.SignatureError, expect.SignatureError)
	}
	if r.DecryptedWith != expect.DecryptedWith {
		t.Errorf("%s: DecryptedWith = \n%v\n but want \n%v", name, r.DecryptedWith, expect.DecryptedWith)
	}
	if r.DecryptedWithKeyId != expect.DecryptedWithKeyId {
		t.Errorf("%s: DecryptedWithKeyId = \n%v\n but want \n%v", name, r.DecryptedWithKeyId, expect.DecryptedWithKeyId)
	}
}

const testKeyId = `B1A8669354153B799F2217BF307215C13DF7A964`

const testPrivateKeyArmored = `-----BEGIN PGP PRIVATE KEY BLOCK-----

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
+5anTnUn
=gemU
-----END PGP PRIVATE KEY BLOCK-----
`

const testPublicKeyArmored = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBF5FJf8BCACvlKhSSsv4P8C3Wbv391SrNUBtFquoMuWKtuCr/Ks6KHuofGLn
bM55uBSQp908aITBDPkaOPsQ3OvwgF7SM8bNIDVpO7FHzCEg2Ysp99iPET/+LsbY
ugc8oYSuvA5aFFIOMYbAbI+HmbIBuCs+xp0AcU1cemAPzPBDCZs4xl5Y+/ce2yQz
ZGK9O/tQQIKoBUOWLo/0byAWyD6Gwn/Le3fVxxK6RPeeizDV6VfzHLxhxBNkMgmd
QUkBkvqF154wYxhzsHn72ushbJpspKz1LQN7d5u6QOq3h2sLwcLbT457qbMfZsZs
HLhoOibOd+yJ7C6TRbbyC4sQRr+K1CNGcvhJABEBAAG0NEpvaG4gRG9lIChUaGlz
IGlzIGEgdGVzdCBrZXkpIDxqb2huLmRvZUBleGFtcGxlLm9yZz6JAU4EEwEIADgW
IQSxqGaTVBU7eZ8iF78wchXBPfepZAUCXkUl/wIbAwULCQgHAgYVCgkICwIEFgID
AQIeAQIXgAAKCRAwchXBPfepZF4iB/49B7q4AfO3xHEa8LK2H+f7Mnm4dRfS2YPo
v2p6TRe1h2DxwpTevNQUhXw2U0nfRIEKBAZqgb7NVktkoh0DWtKatms2yHMAS+ah
lQoHb2gRgXa9M9Tq0x5u9sl0NYnx7Wu5uu6Ybw9luPKoAfO91T0vei0p3eMn3fIV
0O012ITvmgKJPppQDKFJHGZJMbVDO4TNxP89HgyhB41RO7AZadvu73S00x2K6x+O
R4s/++4Y98vScCPm3DUOXeoHXKGqFcNYTxJL9bsE2I0uYgvJSxNoK1dVnmvxp3zz
hcxAdzizgMz0ufY6YLMCjy5MDOzPARkmYPXdkJ6jceOIqGLUw1kquQENBF5FJf8B
CACpsh5cyHB7eEwQvLzJVsXpTW0Rh/Fe36AwC2Vz13WeE6GFrOvw1qATvtYB1919
M4B44YH9J7I5SrFZad86Aw4n5Gi0BwLlGNa/oCMvYzlNHaTXURA271ghJqdZizqV
UETj3WNoaYm4mYMfb0dcayDJvVPWP7InzOsdIRU9WXBUSyVMxNMXccr2UvIuhdPg
lmVT8NtsWR+q8xBoL2Dp0ojYLVD3MlwKe1pE5mEwasYCkWePLWyGdfDW1MhUDsPH
3K1IjpPLWU9FBk8KM4z8WooY9/kyMIyRw39MvOHGfgcFBpiZwlELNZGSFhbRun03
PMk2Qd3k+0FGV1IhFAYsr7QRABEBAAGJATYEGAEIACAWIQSxqGaTVBU7eZ8iF78w
chXBPfepZAUCXkUl/wIbDAAKCRAwchXBPfepZOtqB/9xsGEgQgm70KYID39H91k4
ef/RlpRDY1ndC0MoPfqE03IEXTC/MjtU+ksPKEoZeQsxVaUJ2WBueI5WGJ3Y73pO
HAd7N0SyGHT5s6gK1FSx29be1qiPwUu5KR2jpm3RjgpbymnOWe4C6iiYCFQ85IX+
LzpE+p9bB02PUrmzOb4MBV6E5mg30UjXIX01+bwZq5XSB4/FaUrQOAxLuRvVRjK0
CEcFbPGIlkPSW6s4M9xCC2sQi7caFKVK6Zqf78KbOwAHqfS0x9u2jtTIhsgCjGTI
AOQ5lNwpLEMjwLias6e5sM6hcK9Wo+A9Sw23f8lMau5clOZTJeyAUAff+5anTnUn
=ZjQT
-----END PGP PUBLIC KEY BLOCK-----
`
