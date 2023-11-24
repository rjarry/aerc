package msg

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/log"
	mboxer "git.sr.ht/~rjarry/aerc/worker/mbox"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Pipe struct {
	Background bool   `opt:"-b"`
	Full       bool   `opt:"-m"`
	Part       bool   `opt:"-p"`
	Command    string `opt:"..."`
}

func init() {
	register(Pipe{})
}

func (Pipe) Aliases() []string {
	return []string{"pipe"}
}

func (p Pipe) Execute(args []string) error {
	return p.Run(nil)
}

func (p Pipe) Run(cb func()) error {
	if p.Full && p.Part {
		return errors.New("-m and -p are mutually exclusive")
	}
	name, _, _ := strings.Cut(p.Command, " ")

	provider := app.SelectedTabContent().(app.ProvidesMessage)
	if !p.Full && !p.Part {
		if _, ok := provider.(*app.MessageViewer); ok {
			p.Part = true
		} else if _, ok := provider.(*app.AccountView); ok {
			p.Full = true
		} else {
			return errors.New(
				"Neither -m nor -p specified and cannot infer default")
		}
	}

	doTerm := func(reader io.Reader, name string) {
		cmd := []string{"sh", "-c", p.Command}
		term, err := commands.QuickTerm(cmd, reader)
		if err != nil {
			app.PushError(err.Error())
			return
		}
		if cb != nil {
			last := term.OnClose
			term.OnClose = func(err error) {
				if last != nil {
					last(err)
				}
				cb()
			}
		}
		app.NewTab(term, name)
	}

	doExec := func(reader io.Reader) {
		ecmd := exec.Command("sh", "-c", p.Command)
		pipe, err := ecmd.StdinPipe()
		if err != nil {
			return
		}
		go func() {
			defer log.PanicHandler()

			defer pipe.Close()
			_, err := io.Copy(pipe, reader)
			if err != nil {
				log.Errorf("failed to send data to pipe: %v", err)
			}
		}()
		err = ecmd.Run()
		if err != nil {
			app.PushError(err.Error())
		} else {
			if ecmd.ProcessState.ExitCode() != 0 {
				app.PushError(fmt.Sprintf(
					"%s: completed with status %d", name,
					ecmd.ProcessState.ExitCode()))
			} else {
				app.PushStatus(fmt.Sprintf(
					"%s: completed with status %d", name,
					ecmd.ProcessState.ExitCode()), 10*time.Second)
			}
		}
		if cb != nil {
			cb()
		}
	}

	app.PushStatus("Fetching messages ...", 10*time.Second)

	if p.Full {
		var uids []uint32
		var title string

		h := newHelper()
		store, err := h.store()
		if err != nil {
			if mv, ok := provider.(*app.MessageViewer); ok {
				mv.MessageView().FetchFull(func(reader io.Reader) {
					if p.Background {
						doExec(reader)
					} else {
						doTerm(reader,
							fmt.Sprintf("%s <%s",
								name, title))
					}
				})
				return nil
			}
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
			defer log.PanicHandler()

			select {
			case <-done:
				break
			case <-time.After(30 * time.Second):
				// TODO: find a better way to determine if store.FetchFull()
				// has finished with some errors.
				app.PushError("Failed to fetch all messages")
				if len(messages) == 0 {
					return
				}
			}

			is_git_patches := false
			for _, msg := range messages {
				info := store.Messages[msg.Content.Uid]
				if info != nil && patchSeriesRe.MatchString(info.Envelope.Subject) {
					is_git_patches = true
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
					return infoi.Envelope.Subject < infoj.Envelope.Subject
				})
			}

			reader := newMessagesReader(messages, len(messages) > 1)
			if p.Background {
				doExec(reader)
			} else {
				doTerm(reader, fmt.Sprintf("%s <%s", name, title))
			}
		}()
	} else if p.Part {
		mv, ok := provider.(*app.MessageViewer)
		if !ok {
			return fmt.Errorf("can only pipe message part from a message view")
		}
		part := provider.SelectedMessagePart()
		if part == nil {
			return fmt.Errorf("could not fetch message part")
		}
		mv.MessageView().FetchBodyPart(part.Index, func(reader io.Reader) {
			if p.Background {
				doExec(reader)
			} else {
				name := fmt.Sprintf("%s <%s/[%d]",
					name, part.Msg.Envelope.Subject, part.Index)
				doTerm(reader, name)
			}
		})
	}
	if store := provider.Store(); store != nil {
		store.Marker().ClearVisualMark()
	}
	return nil
}

func newMessagesReader(messages []*types.FullMessage, useMbox bool) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		defer log.PanicHandler()
		defer pw.Close()
		for _, msg := range messages {
			var err error
			if useMbox {
				err = mboxer.Write(pw, msg.Content.Reader, "", time.Now())
			} else {
				_, err = io.Copy(pw, msg.Content.Reader)
			}
			if err != nil {
				log.Warnf("failed to write data: %v", err)
			}
		}
	}()
	return pr
}

var patchSeriesRe = regexp.MustCompile(
	`^.*\[(RFC )?PATCH( [^\]]+)? \d+/\d+] .+$`,
)
