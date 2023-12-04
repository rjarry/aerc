package msg

import (
	"errors"
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
)

type Envelope struct {
	Header bool   `opt:"-h"`
	Format string `opt:"-s" default:"%-20.20s: %s"`
}

func init() {
	commands.Register(Envelope{})
}

func (Envelope) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (Envelope) Aliases() []string {
	return []string{"envelope"}
}

func (e Envelope) Execute(args []string) error {
	provider, ok := app.SelectedTabContent().(app.ProvidesMessages)
	if !ok {
		return fmt.Errorf("current tab does not implement app.ProvidesMessage interface")
	}

	acct := provider.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	var list []string
	if msg, err := provider.SelectedMessage(); err != nil {
		return err
	} else {
		if msg != nil {
			if e.Header {
				list = parseHeader(msg, e.Format)
			} else {
				list = parseEnvelope(msg, e.Format,
					acct.UiConfig().TimestampFormat)
			}
		} else {
			return fmt.Errorf("Selected message is empty.")
		}
	}

	n := len(list)
	app.AddDialog(app.NewDialog(
		app.NewListBox(
			"Message Envelope. Press <Esc> or <Enter> to close. "+
				"Start typing to filter.",
			list,
			app.SelectedAccountUiConfig(),
			func(_ string) {
				app.CloseDialog()
			},
		),
		// start pos on screen
		func(h int) int {
			if n < h/8*6 {
				return h/2 - n/2 - 1
			}
			return h / 8
		},
		// dialog height
		func(h int) int {
			if n < h/8*6 {
				return n + 2
			}
			return h / 8 * 6
		},
	))

	return nil
}

func parseEnvelope(msg *models.MessageInfo, fmtStr, fmtTime string,
) (result []string) {
	if envlp := msg.Envelope; envlp != nil {
		addStr := func(key, text string) {
			result = append(result, fmt.Sprintf(fmtStr, key, text))
		}
		addAddr := func(key string, ls []*mail.Address) {
			for _, l := range ls {
				result = append(result,
					fmt.Sprintf(fmtStr, key,
						format.AddressForHumans(l)))
			}
		}

		addStr("Date", envlp.Date.Format(fmtTime))
		addStr("Subject", envlp.Subject)
		addStr("Message-Id", envlp.MessageId)

		addAddr("From", envlp.From)
		addAddr("To", envlp.To)
		addAddr("ReplyTo", envlp.ReplyTo)
		addAddr("Cc", envlp.Cc)
		addAddr("Bcc", envlp.Bcc)
	}
	return
}

func parseHeader(msg *models.MessageInfo, fmtStr string) (result []string) {
	if h := msg.RFC822Headers; h != nil {
		hf := h.Fields()
		for hf.Next() {
			text, err := hf.Text()
			if err != nil {
				log.Errorf(err.Error())
				text = hf.Value()
			}
			result = append(result,
				headerExpand(fmtStr, hf.Key(), text)...)
		}
	}
	return
}

func headerExpand(fmtStr, key, text string) []string {
	var result []string
	switch strings.ToLower(key) {
	case "to", "from", "bcc", "cc":
		for _, item := range strings.Split(text, ",") {
			result = append(result, fmt.Sprintf(fmtStr, key,
				strings.TrimSpace(item)))
		}
	default:
		result = append(result, fmt.Sprintf(fmtStr, key, text))
	}
	return result
}
