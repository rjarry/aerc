package send

import (
	"fmt"
	"io"
	"net/url"
	"os/exec"

	"git.sr.ht/~rjarry/go-opt"
	"github.com/emersion/go-message/mail"
	"github.com/pkg/errors"
)

type sendmailSender struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func (s *sendmailSender) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

func (s *sendmailSender) Close() error {
	se := s.stdin.Close()
	ce := s.cmd.Wait()
	if se != nil {
		return se
	}
	return ce
}

func newSendmailSender(uri *url.URL, rcpts []*mail.Address) (io.WriteCloser, error) {
	args := opt.SplitArgs(uri.Path)
	if len(args) == 0 {
		return nil, fmt.Errorf("no command specified")
	}
	bin := args[0]
	rs := make([]string, len(rcpts))
	for i := range rcpts {
		rs[i] = rcpts[i].Address
	}
	args = append(args[1:], rs...)
	cmd := exec.Command(bin, args...)
	s := &sendmailSender{cmd: cmd}
	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "cmd.StdinPipe")
	}
	err = s.cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, "cmd.Start")
	}
	return s, nil
}
