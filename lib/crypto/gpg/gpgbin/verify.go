package gpgbin

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"git.sr.ht/~rjarry/aerc/models"
)

// Verify runs gpg --verify. If s is not nil, then gpg interprets the
// arguments as a detached signature
func Verify(m io.Reader, s io.Reader) (*models.MessageDetails, error) {
	args := []string{"--verify"}
	if s != nil {
		// Detached sig, save the sig to a tmp file and send msg over stdin
		sig, err := ioutil.TempFile("", "sig")
		if err != nil {
			return nil, err
		}
		_, _ = io.Copy(sig, s)
		sig.Close()
		defer os.Remove(sig.Name())
		args = append(args, sig.Name(), "-")
	}
	orig, err := ioutil.ReadAll(m)
	if err != nil {
		return nil, err
	}
	g := newGpg(bytes.NewReader(orig), args)
	err = g.cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("gpg: failed to run verification: %w", err)
	}

	out := bytes.NewReader(g.stdout.Bytes())
	md := new(models.MessageDetails)
	err = parse(out, md)
	if err != nil {
		return nil, fmt.Errorf("gpg: failed to parse result: %w", err)
	}

	md.Body = bytes.NewReader(orig)

	return md, nil
}
