package msg

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-message/mail"
)

type forward struct {
	AttachAll  bool     `opt:"-A"`
	AttachFull bool     `opt:"-F"`
	Edit       bool     `opt:"-e"`
	NoEdit     bool     `opt:"-E"`
	Template   string   `opt:"-T" complete:"CompleteTemplate"`
	To         []string `opt:"..." required:"false" complete:"CompleteTo"`
}

func init() {
	commands.Register(forward{})
}

func (forward) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (forward) Aliases() []string {
	return []string{"forward"}
}

func (*forward) CompleteTemplate(arg string) []string {
	return commands.GetTemplates(arg)
}

func (*forward) CompleteTo(arg string) []string {
	return commands.GetAddress(arg)
}

func (f forward) Execute(args []string) error {
	if f.AttachAll && f.AttachFull {
		return errors.New("Options -A and -F are mutually exclusive")
	}
	editHeaders := (config.Compose.EditHeaders || f.Edit) && !f.NoEdit

	widget := app.SelectedTabContent().(app.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	log.Debugf("Forwarding email <%s>", msg.Envelope.MessageId)

	h := &mail.Header{}
	subject := "Fwd: " + msg.Envelope.Subject
	h.SetSubject(subject)

	var tolist []*mail.Address
	to := strings.Join(f.To, ", ")
	if strings.Contains(to, "@") {
		tolist, err = mail.ParseAddressList(to)
		if err != nil {
			return fmt.Errorf("invalid to address(es): %w", err)
		}
	}
	if len(tolist) > 0 {
		h.SetAddressList("to", tolist)
	}

	original := models.OriginalMail{
		From:          format.FormatAddresses(msg.Envelope.From),
		Date:          msg.Envelope.Date,
		RFC822Headers: msg.RFC822Headers,
	}

	addTab := func() (*app.Composer, error) {
		composer, err := app.NewComposer(acct,
			acct.AccountConfig(), acct.Worker(), editHeaders,
			f.Template, h, &original, nil)
		if err != nil {
			app.PushError("Error: " + err.Error())
			return nil, err
		}

		composer.Tab = app.NewTab(composer, subject)
		if !h.Has("to") {
			composer.FocusEditor("to")
		} else {
			composer.FocusTerminal()
		}
		return composer, nil
	}

	mv, isMsgViewer := widget.(*app.MessageViewer)
	store := widget.Store()
	noStore := store == nil
	if noStore && !isMsgViewer {
		return errors.New("Cannot perform action. Messages still loading")
	}

	if f.AttachFull {
		tmpDir, err := os.MkdirTemp("", "aerc-tmp-attachment")
		if err != nil {
			return err
		}
		tmpFileName := path.Join(tmpDir,
			strings.ReplaceAll(fmt.Sprintf("%s.eml", msg.Envelope.Subject), "/", "-"))

		var fetchFull func(func(io.Reader))

		if isMsgViewer {
			fetchFull = mv.MessageView().FetchFull
		} else {
			fetchFull = func(cb func(io.Reader)) {
				store.FetchFull([]uint32{msg.Uid}, func(fm *types.FullMessage) {
					if fm == nil || (fm != nil && fm.Content == nil) {
						return
					}
					cb(fm.Content.Reader)
				})
			}
		}

		fetchFull(func(r io.Reader) {
			tmpFile, err := os.Create(tmpFileName)
			if err != nil {
				log.Warnf("failed to create temporary attachment: %v", err)
				_, err = addTab()
				if err != nil {
					log.Warnf("failed to add tab: %v", err)
				}
				return
			}

			defer tmpFile.Close()
			_, err = io.Copy(tmpFile, r)
			if err != nil {
				log.Warnf("failed to write to tmpfile: %v", err)
				return
			}
			composer, err := addTab()
			if err != nil {
				return
			}
			composer.AddAttachment(tmpFileName)
			composer.OnClose(func(c *app.Composer) {
				if c.Sent() {
					store.Forwarded([]uint32{msg.Uid}, true, nil)
				}
				os.RemoveAll(tmpDir)
			})
		})
	} else {
		if f.Template == "" {
			f.Template = config.Templates.Forwards
		}

		var fetchBodyPart func([]int, func(io.Reader))

		if isMsgViewer {
			fetchBodyPart = mv.MessageView().FetchBodyPart
		} else {
			fetchBodyPart = func(part []int, cb func(io.Reader)) {
				store.FetchBodyPart(msg.Uid, part, cb)
			}
		}

		if crypto.IsEncrypted(msg.BodyStructure) && !isMsgViewer {
			return fmt.Errorf("message is encrypted. " +
				"can only forward from the message viewer")
		}

		part := getMessagePart(msg, widget)
		if part == nil {
			part = lib.FindFirstNonMultipart(msg.BodyStructure, nil)
			// if it's still nil here, we don't have a multipart msg, that's fine
		}

		err = addMimeType(msg, part, &original)
		if err != nil {
			return err
		}

		fetchBodyPart(part, func(reader io.Reader) {
			buf := new(bytes.Buffer)
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				buf.WriteString(scanner.Text() + "\n")
			}
			original.Text = buf.String()

			// create composer
			composer, err := addTab()
			if err != nil {
				return
			}

			composer.OnClose(func(c *app.Composer) {
				if c.Sent() {
					store.Forwarded([]uint32{msg.Uid}, true, nil)
				}
			})

			// add attachments
			if f.AttachAll {
				var mu sync.Mutex
				parts := lib.FindAllNonMultipart(msg.BodyStructure, nil, nil)
				for _, p := range parts {
					if lib.EqualParts(p, part) {
						continue
					}
					bs, err := msg.BodyStructure.PartAtIndex(p)
					if err != nil {
						log.Errorf("cannot get PartAtIndex %v: %v", p, err)
						continue
					}
					fetchBodyPart(p, func(reader io.Reader) {
						mime := bs.FullMIMEType()
						params := lib.SetUtf8Charset(bs.Params)
						name, ok := params["name"]
						if !ok {
							name = fmt.Sprintf("%s_%s_%d", bs.MIMEType, bs.MIMESubType, rand.Uint64())
						}
						mu.Lock()
						err := composer.AddPartAttachment(name, mime, params, reader)
						mu.Unlock()
						if err != nil {
							log.Errorf(err.Error())
							app.PushError(err.Error())
						}
					})
				}
			}
		})
	}
	return nil
}
