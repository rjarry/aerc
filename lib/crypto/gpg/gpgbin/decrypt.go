package gpgbin

import (
	"bytes"
	"io"
	"io/ioutil"

	"git.sr.ht/~rjarry/aerc/models"
)

// Decrypt runs gpg --decrypt on the contents of r. If the packet is signed,
// the signature is also verified
func Decrypt(r io.Reader) (*models.MessageDetails, error) {
	md := new(models.MessageDetails)
	orig, err := ioutil.ReadAll(r)
	if err != nil {
		return md, err
	}
	args := []string{"--decrypt"}
	g := newGpg(bytes.NewReader(orig), args)
	err = g.cmd.Run()
	if err != nil {
		err = parseError(g.stderr.String())
		switch GPGErrors[err.Error()] {
		case ERROR_NO_PGP_DATA_FOUND:
			md.Body = bytes.NewReader(orig)
			return md, nil
		default:
			return nil, err
		}
	}
	outRdr := bytes.NewReader(g.stdout.Bytes())
	parse(outRdr, md)
	return md, nil
}
