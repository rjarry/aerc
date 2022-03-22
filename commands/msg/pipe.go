package msg

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"time"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"

	"git.sr.ht/~sircmpwn/getopt"
)

type Pipe struct{}

func init() {
	register(Pipe{})
}

func (Pipe) Aliases() []string {
	return []string{"pipe"}
}

func (Pipe) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Pipe) Execute(aerc *widgets.Aerc, args []string) error {
	var (
		background bool
		pipeFull   bool
		pipePart   bool
	)
	// TODO: let user specify part by index or preferred mimetype
	opts, optind, err := getopt.Getopts(args, "bmp")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'b':
			background = true
		case 'm':
			if pipePart {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipeFull = true
		case 'p':
			if pipeFull {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipePart = true
		}
	}
	cmd := args[optind:]
	if len(cmd) == 0 {
		return errors.New("Usage: pipe [-mp] <cmd> [args...]")
	}

	provider := aerc.SelectedTab().(widgets.ProvidesMessage)
	if !pipeFull && !pipePart {
		if _, ok := provider.(*widgets.MessageViewer); ok {
			pipePart = true
		} else if _, ok := provider.(*widgets.AccountView); ok {
			pipeFull = true
		} else {
			return errors.New(
				"Neither -m nor -p specified and cannot infer default")
		}
	}

	doTerm := func(reader io.Reader, name string) {
		term, err := commands.QuickTerm(aerc, cmd, reader)
		if err != nil {
			aerc.PushError(err.Error())
			return
		}
		aerc.NewTab(term, name)
	}

	doExec := func(reader io.Reader) {
		ecmd := exec.Command(cmd[0], cmd[1:]...)
		pipe, err := ecmd.StdinPipe()
		if err != nil {
			return
		}
		go func() {
			defer logging.PanicHandler()

			defer pipe.Close()
			io.Copy(pipe, reader)
		}()
		err = ecmd.Run()
		if err != nil {
			aerc.PushError(err.Error())
		} else {
			if ecmd.ProcessState.ExitCode() != 0 {
				aerc.PushError(fmt.Sprintf(
					"%s: completed with status %d", cmd[0],
					ecmd.ProcessState.ExitCode()))
			} else {
				aerc.PushStatus(fmt.Sprintf(
					"%s: completed with status %d", cmd[0],
					ecmd.ProcessState.ExitCode()), 10*time.Second)
			}
		}
	}

	if pipeFull {
		var uids []uint32
		var title string

		h := newHelper(aerc)
		store, err := h.store()
		if err != nil {
			return err
		}
		uids, err = h.markedOrSelectedUids()
		if err != nil {
			return err
		}

		if len(uids) == 1 {
			info := store.Messages[uids[0]]
			if info != nil {
				envelope := info.Envelope
				if envelope != nil {
					title = envelope.Subject
				}
			}
		}
		if title == "" {
			title = fmt.Sprintf("%d messages", len(uids))
		}

		var messages []*types.FullMessage
		done := make(chan bool, 1)

		store.FetchFull(uids, func(fm *types.FullMessage) {
			messages = append(messages, fm)
			if len(messages) == len(uids) {
				done <- true
			}
		})

		go func() {
			defer logging.PanicHandler()

			select {
			case <-done:
				break
			case <-time.After(30 * time.Second):
				// TODO: find a better way to determine if store.FetchFull()
				// has finished with some errors.
				aerc.PushError("Failed to fetch all messages")
				if len(messages) == 0 {
					return
				}
			}

			// Sort all messages by increasing Message-Id header.
			// This will ensure that patch series are applied in order.
			sort.Slice(messages, func(i, j int) bool {
				infoi := store.Messages[messages[i].Content.Uid]
				infoj := store.Messages[messages[j].Content.Uid]
				if infoi == nil || infoj == nil {
					return false
				}
				return infoi.Envelope.MessageId < infoj.Envelope.MessageId
			})

			reader := newMessagesReader(messages)
			if background {
				doExec(reader)
			} else {
				doTerm(reader, fmt.Sprintf("%s <%s", cmd[0], title))
			}
		}()
	} else if pipePart {
		p := provider.SelectedMessagePart()
		if p == nil {
			return fmt.Errorf("could not fetch message part")
		}
		store := provider.Store()
		store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
			if background {
				doExec(reader)
			} else {
				name := fmt.Sprintf("%s <%s/[%d]",
					cmd[0], p.Msg.Envelope.Subject, p.Index)
				doTerm(reader, name)
			}
		})
	}

	return nil
}

// The actual sender address does not matter, nor does the date. This is mostly indended
// for git am which requires separators to look like something valid.
// https://github.com/git/git/blame/v2.35.1/builtin/mailsplit.c#L15-L44
var mboxSeparator []byte = []byte("From ???@??? Tue Jun 23 16:32:49 1981\n")

type messagesReader struct {
	messages        []*types.FullMessage
	mbox            bool
	separatorNeeded bool
}

func newMessagesReader(messages []*types.FullMessage) io.Reader {
	needMboxSeparator := len(messages) > 1
	return &messagesReader{messages, needMboxSeparator, needMboxSeparator}
}

func (mr *messagesReader) Read(p []byte) (n int, err error) {
	for len(mr.messages) > 0 {
		if mr.separatorNeeded {
			offset := copy(p, mboxSeparator)
			n, err = mr.messages[0].Content.Reader.Read(p[offset:])
			n += offset
			mr.separatorNeeded = false
		} else {
			n, err = mr.messages[0].Content.Reader.Read(p)
		}
		if err == io.EOF {
			mr.messages = mr.messages[1:]
			mr.separatorNeeded = mr.mbox
		}
		if n > 0 || err != io.EOF {
			if err == io.EOF && len(mr.messages) > 0 {
				// Don't return EOF yet. More messages remain.
				err = nil
			}
			return n, err
		}
	}
	return 0, io.EOF
}
