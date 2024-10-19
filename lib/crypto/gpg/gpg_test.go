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

mQENBGcUGPEBCACox9bw5BiN9M+1qVtU90bkHl5xzPDl8SqX/2ieYSx0ZfUpmRAH
9EbW4j54cTFM6mX18Yv2LRWQhHjzslPietJ1Lb3PGY2ffDDxJsq/uQHK/ztqePc7
omJJjUuF5D7BjuOq/MFyu7dWSCXOrj8soY9HIS96pPNTF9ykLDhqKWIqGA7pORKk
RFczMLmEojLKefHvgtp9ikNNbIJyq/P5hNHr/DfC7rFaMTrXNc2xP2MD7MYNdVmT
N2NN/X676rTsu8ltUi96F5PR33mGez6Z66yMjJf863bd+muq8552ExoQGQ/uGo5y
wvwoEOF7hx1Z6JYl56hAICXPL/ZOZTPdBf+9ABEBAAG0NEphbmUgRG9lIChUaGlz
IGlzIGEgdGVzdCBrZXkpIDxqYW5lLmRvZUBleGFtcGxlLm9yZz6JAVEEEwEIADsW
IQSoQ3iEudN9vdxgn6xy8nGZUc/d5AUCZxQY8QIbAwULCQgHAgIiAgYVCgkICwIE
FgIDAQIeBwIXgAAKCRBy8nGZUc/d5ConB/9Z39ufzGmplm0m9ylN+x8iNYJJ5rk6
WhnwDsKSEDPoYnSUuESQ7zxhPkqr2amgAcFWba6vm+GvdFBB+y8JzSGIBmNmQfuw
dtBd5EI+cTSTzuXo4NXR7TrMJGPP8IvJNSrliG61JnW3kcz9U9dywum+XF57+2X1
KCt3npJI64sMX39QZ1ReaRbKWrKcBdCWZqW79KbFn4yl4ooMS9aKggQQP91feMA9
dP3onL+TWLRKVMQ657OngTKi8rIez+RasRmVV3Av+GMl0Tdcg3sWHrlliBexmC/X
mHzbl/PR8HAjWxie+pObGPz1aodJpeI0Lr5LQgJxZtx49kov9Ua5xVUxuQENBGcU
GPEBCACmVEII6Igka7AVqCrUrdRonSzuelT6X6/VToBoJMER7q5MENtqWd0iby4N
kIJxaJQFyXY7mYyZqf2aRbCu+cvh/F77iSZEOzNoJuut5sjPg7MM+s/9GRlYboq9
RGqDJwoT7+k6cdUJON5UPvdJj8GnFGGu9ZFs/cOz2psggzfeV4YbTKXzFm2yKMpx
LdeBeLXLYG46d0ChZMmKyBLLJWtUb71MU2TTWyrmtDoN02bxDQpAeJu+3Qp6lq+/
CGe5f407jkx2PDKvV6HkuYzjs8apVFVZsBkDlhkaX5YdFI2r1TxIbxC9k2UG9VLJ
lGNeqO3iUCsjuKd7iaiLGGBIeqKnABEBAAGJATYEGAEIACAWIQSoQ3iEudN9vdxg
n6xy8nGZUc/d5AUCZxQY8QIbDAAKCRBy8nGZUc/d5OxbB/sEqrdtCMFrXLOU7dur
or1lfrlYaOIaOup+/SnTSi688O0ixZ2XjV7CW3z1E8JjWAVsQPdfpC2QOZATWZ/q
ZMuEMwNpzhCVZDwBJR7nw+Pv/xFv9DvLEiJYHCyBrQtQ6vopG0t2yxJ4R/R48fQC
m2xT54mb4flIV/C8zRy3eK2wY/kR5FVxnLwwFlYayR7+wuLTiHqqxRyeZA3hQcF3
YDOgvRu3YzmESPtIBI6iNphfSSAAtkUqNJnwPAIxyky8xEInUZ7maOADRWgEH8uG
+1FjPta6cgZ1tJzFtJ7Bwa2///UAp7BQqDl7DyMQAfOZGkUI9mqEXdra4YqMv5X0
Y2UQ
=QL1U
-----END PGP PUBLIC KEY BLOCK-----
`

const testOwnertrust = "B1A8669354153B799F2217BF307215C13DF7A964:6:\n"
