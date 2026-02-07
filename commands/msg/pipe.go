package msg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	cryptoutil "git.sr.ht/~rjarry/aerc/lib/crypto/util"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	mboxer "git.sr.ht/~rjarry/aerc/worker/mbox"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Pipe struct {
	Background bool   `opt:"-b" desc:"Run the command in the background."`
	Silent     bool   `opt:"-s" desc:"Silently close the terminal tab after the command exits."`
	Full       bool   `opt:"-m" desc:"Pipe the full message."`
	Decrypt    bool   `opt:"-d" desc:"Decrypt the full message before piping."`
	Part       bool   `opt:"-p" desc:"Only pipe the selected message part."`
	Command    string `opt:"..."`
}

func init() {
	commands.Register(Pipe{})
}

func (Pipe) Description() string {
	return "Pipe the selected message(s) into the given shell command."
}

func (Pipe) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER | commands.COMPOSE_REVIEW
}

func (Pipe) Aliases() []string {
	return []string{"pipe"}
}

func (p Pipe) Execute(args []string) error {
	return p.Run(nil)
}

// doTerm executes the command in an interactive terminal tab
func doTerm(command string, reader io.Reader, name string, silent bool, cb func()) {
	cmd := []string{"sh", "-c", command}
	term, err := commands.QuickTerm(cmd, reader, silent)
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

// doExec executes the command in the background
func doExec(command string, reader io.Reader, name string, cb func()) {
	ecmd := exec.Command("sh", "-c", command)
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

func (p Pipe) Run(cb func()) error {
	if p.Decrypt {
		// Decrypt implies fetching the full message
		p.Full = true
	}
	if p.Full && p.Part {
		return errors.New("-m and -p are mutually exclusive")
	}
	name, _, _ := strings.Cut(p.Command, " ")

	// Special handling for Composer in review mode
	if composer, ok := app.SelectedTabContent().(*app.Composer); ok && composer.Bindings() == "compose::review" {
		// Get the message content
		header, err := composer.PrepareHeader()
		if err != nil {
			return errors.Wrap(err, "PrepareHeader")
		}

		pr, pw := io.Pipe()
		go func() {
			defer log.PanicHandler()
			defer pw.Close()
			err := composer.WriteMessage(header, pw)
			if err != nil {
				log.Errorf("failed to write message: %v", err)
			}
		}()

		if p.Background {
			doExec(p.Command, pr, name, cb)
		} else {
			doTerm(p.Command, pr, fmt.Sprintf("%s <review>", name), p.Silent, cb)
		}
		return nil
	}

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

	app.PushStatus("Fetching messages ...", 10*time.Second)

	if p.Full {
		var uids []models.UID
		var title string

		h := newHelper()
		store, err := h.store()
		if err != nil {
			if mv, ok := provider.(*app.MessageViewer); ok {
				mv.MessageView().FetchFull(func(reader io.Reader) {
					if p.Background {
						doExec(p.Command, reader, name, cb)
					} else {
						doTerm(p.Command, reader,
							fmt.Sprintf("%s <%s",
								name, title), p.Silent, cb)
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
		var errors []error
		done := make(chan bool, 1)

		store.FetchFull(context.TODO(), uids, func(fm *types.FullMessage) {
			if p.Decrypt {
				info := store.Messages[fm.Content.Uid]
				if info == nil {
					goto addMessage
				}
				var buf bytes.Buffer
				cleartext, err := cryptoutil.Cleartext(
					io.TeeReader(fm.Content.Reader, &buf),
					info.RFC822Headers.Copy(),
				)
				if err != nil {
					log.Warnf("continue encrypted: %v", err)
					fm.Content.Reader = bytes.NewReader(buf.Bytes())
				} else {
					fm.Content.Reader = bytes.NewReader(cleartext)
				}
			}
		addMessage:
			info := store.Messages[fm.Content.Uid]
			switch {
			case info != nil && info.Envelope != nil:
				messages = append(messages, fm)
			case info != nil && info.Error != nil:
				app.PushError(info.Error.Error())
				errors = append(errors, info.Error)
			default:
				err := fmt.Errorf("%v nil info", fm.Content.Uid)
				app.PushError(err.Error())
				errors = append(errors, err)
			}
			if len(messages)+len(errors) == len(uids) {
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
				if info == nil || info.Envelope == nil {
					continue
				}
				if patchSeriesRe.MatchString(info.Envelope.Subject) {
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
					if infoi == nil || infoi.Envelope == nil ||
						infoj == nil || infoj.Envelope == nil {
						return false
					}
					return infoi.Envelope.Subject < infoj.Envelope.Subject
				})
			}

			reader := newMessagesReader(messages, len(messages) > 1)
			if p.Background {
				doExec(p.Command, reader, name, cb)
			} else {
				doTerm(p.Command, reader, fmt.Sprintf("%s <%s", name, title), p.Silent, cb)
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
				doExec(p.Command, reader, name, cb)
			} else {
				termName := fmt.Sprintf("%s <%s/[%d]",
					name, part.Msg.Envelope.Subject, part.Index)
				doTerm(p.Command, reader, termName, p.Silent, cb)
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
