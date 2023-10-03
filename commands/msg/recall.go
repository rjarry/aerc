package msg

import (
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	_ "github.com/emersion/go-message/charset"
	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Recall struct {
	Force  bool `opt:"-f"`
	Edit   bool `opt:"-e"`
	NoEdit bool `opt:"-E"`
}

func init() {
	register(Recall{})
}

func (Recall) Aliases() []string {
	return []string{"recall"}
}

func (Recall) Complete(args []string) []string {
	return nil
}

func (r Recall) Execute(args []string) error {
	editHeaders := (config.Compose.EditHeaders || r.Edit) && !r.NoEdit

	widget := app.SelectedTabContent().(app.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if acct.SelectedDirectory() != acct.AccountConfig().Postpone && !r.Force {
		return errors.New("Use -f to recall from outside the " +
			acct.AccountConfig().Postpone + " directory.")
	}
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	msgInfo, err := widget.SelectedMessage()
	if err != nil {
		return errors.Wrap(err, "Recall failed")
	}
	log.Debugf("Recalling message <%s>", msgInfo.Envelope.MessageId)

	addTab := func(composer *app.Composer) {
		subject := msgInfo.Envelope.Subject
		if subject == "" {
			subject = "Recalled email"
		}
		composer.Tab = app.NewTab(composer, subject)
		composer.OnClose(func(composer *app.Composer) {
			worker := composer.Worker()
			uids := []uint32{msgInfo.Uid}

			deleteMessage := func() {
				worker.PostAction(&types.DeleteMessages{
					Uids: uids,
				}, func(msg types.WorkerMessage) {
					switch msg := msg.(type) {
					case *types.Done:
						app.PushStatus("Recalled message deleted", 10*time.Second)
					case *types.Error:
						app.PushError(msg.Error.Error())
					}
				})
			}

			if composer.Sent() || composer.Postponed() {
				deleteMessage()
			}
		})
	}

	lib.NewMessageStoreView(msgInfo, acct.UiConfig().AutoMarkRead,
		store, app.CryptoProvider(), app.DecryptKeys,
		func(msg lib.MessageView, err error) {
			if err != nil {
				app.PushError(err.Error())
				return
			}
			var path []int
			if len(msg.BodyStructure().Parts) != 0 {
				path = lib.FindPlaintext(msg.BodyStructure(), path)
			}

			msg.FetchBodyPart(path, func(reader io.Reader) {
				composer, err := app.NewComposer(acct,
					acct.AccountConfig(), acct.Worker(), editHeaders,
					"", msgInfo.RFC822Headers, nil, reader)
				if err != nil {
					app.PushError(err.Error())
					return
				}
				if md := msg.MessageDetails(); md != nil {
					if md.IsEncrypted {
						composer.SetEncrypt(md.IsEncrypted)
					}
					if md.IsSigned {
						err = composer.SetSign(md.IsSigned)
						if err != nil {
							log.Warnf("failed to set signed state: %v", err)
						}
					}
				}

				// add attachements if present
				var mu sync.Mutex
				parts := lib.FindAllNonMultipart(msg.BodyStructure(), nil, nil)
				for _, p := range parts {
					if lib.EqualParts(p, path) {
						continue
					}
					bs, err := msg.BodyStructure().PartAtIndex(p)
					if err != nil {
						log.Warnf("cannot get PartAtIndex %v: %v", p, err)
						continue
					}
					msg.FetchBodyPart(p, func(reader io.Reader) {
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

				if r.Force {
					composer.SetRecalledFrom(acct.SelectedDirectory())
				}

				// focus the terminal since the header fields are likely already done
				composer.FocusTerminal()
				addTab(composer)
			})
		})

	return nil
}
