package msg

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"time"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/widgets"
	mboxer "git.sr.ht/~rjarry/aerc/worker/mbox"
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

	provider := aerc.SelectedTabContent().(widgets.ProvidesMessage)
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
			_, err := io.Copy(pipe, reader)
			if err != nil {
				logging.Errorf("failed to send data to pipe: %w", err)
			}
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

			is_git_patches := true
			for _, msg := range messages {
				info := store.Messages[msg.Content.Uid]
				if info == nil || !gitMessageIdRe.MatchString(info.Envelope.MessageId) {
					is_git_patches = false
					break
				}
			}
			if is_git_patches {
				// Sort all messages by increasing Message-Id header.
				// This will ensure that patch series are applied in order.
				sort.Slice(messages, func(i, j int) bool {
					infoi := store.Messages[messages[i].Content.Uid]
					infoj := store.Messages[messages[j].Content.Uid]
					if infoi == nil || infoj == nil {
						return false
					}
					msgidi := padGitMessageId(infoi.Envelope.MessageId)
					msgidj := padGitMessageId(infoj.Envelope.MessageId)
					return msgidi < msgidj
				})
			}

			reader := newMessagesReader(messages, len(messages) > 1)
			if background {
				doExec(reader)
			} else {
				doTerm(reader, fmt.Sprintf("%s <%s", cmd[0], title))
			}
		}()
	} else if pipePart {
		mv, ok := provider.(*widgets.MessageViewer)
		if !ok {
			return fmt.Errorf("can only pipe message part from a message view")
		}
		p := provider.SelectedMessagePart()
		if p == nil {
			return fmt.Errorf("could not fetch message part")
		}
		mv.MessageView().FetchBodyPart(p.Index, func(reader io.Reader) {
			if background {
				doExec(reader)
			} else {
				name := fmt.Sprintf("%s <%s/[%d]",
					cmd[0], p.Msg.Envelope.Subject, p.Index)
				doTerm(reader, name)
			}
		})
	}
	provider.Store().Marker().ClearVisualMark()
	return nil
}

func newMessagesReader(messages []*types.FullMessage, useMbox bool) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for _, msg := range messages {
			var err error
			if useMbox {
				err = mboxer.Write(pw, msg.Content.Reader, "", time.Now())
			} else {
				_, err = io.Copy(pw, msg.Content.Reader)
			}
			if err != nil {
				logging.Warnf("failed to write data: %v", err)
			}
		}
	}()
	return pr
}

var gitMessageIdRe = regexp.MustCompile(`^(\d+\.\d+)-(\d+)-(.+)$`)

// Git send-email Message-Id headers have the following format:
//
//	DATETIME.PID-NUM-COMMITTER
//
// Return a copy of the message id with NUM zero-padded to three characters.
func padGitMessageId(msgId string) string {
	matches := gitMessageIdRe.FindStringSubmatch(msgId)
	if matches == nil {
		return msgId
	}
	number, err := strconv.Atoi(matches[2])
	if err != nil {
		return msgId
	}
	return fmt.Sprintf("%s-%03d-%s", matches[1], number, matches[3])
}
