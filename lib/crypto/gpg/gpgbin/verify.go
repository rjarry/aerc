package gpgbin

import (
	"bytes"
	"io"
	"os"

	"git.sr.ht/~rjarry/aerc/models"
)

// Verify runs gpg --verify. If s is not nil, then gpg interprets the
// arguments as a detached signature
func Verify(m io.Reader, s io.Reader) (*models.MessageDetails, error) {
	args := []string{"--verify"}
	if s != nil {
		// Detached sig, save the sig to a tmp file and send msg over stdin
		sig, err := os.CreateTemp("", "sig")
		if err != nil {
			return nil, err
		}
		_, _ = io.Copy(sig, s)
		sig.Close()
		defer os.Remove(sig.Name())
		args = append(args, sig.Name(), "-")
	}
	orig, err := io.ReadAll(m)
	if err != nil {
		return nil, err
	}
	g := newGpg(bytes.NewReader(orig), args)
	_ = g.cmd.Run()

	out := bytes.NewReader(g.stdout.Bytes())
	md := new(models.MessageDetails)
	_ = parse(out, md)

	md.Body = bytes.NewReader(orig)

	return md, nil
}
