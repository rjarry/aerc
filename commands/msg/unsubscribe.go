package msg

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"github.com/emersion/go-message/mail"
)

// Unsubscribe helps people unsubscribe from mailing lists by way of the
// List-Unsubscribe header.
type Unsubscribe struct{}

func init() {
	register(Unsubscribe{})
}

// Aliases returns a list of aliases for the :unsubscribe command
func (Unsubscribe) Aliases() []string {
	return []string{"unsubscribe"}
}

// Complete returns a list of completions
func (Unsubscribe) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

// Execute runs the Unsubscribe command
func (Unsubscribe) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: unsubscribe")
	}
	widget := aerc.SelectedTabContent().(widgets.ProvidesMessage)
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	headers := msg.RFC822Headers
	if !headers.Has("list-unsubscribe") {
		return errors.New("No List-Unsubscribe header found")
	}
	text, err := headers.Text("list-unsubscribe")
	if err != nil {
		return err
	}
	methods := parseUnsubscribeMethods(text)
	if len(methods) == 0 {
		return fmt.Errorf("no methods found to unsubscribe")
	}
	log.Debugf("unsubscribe: found %d methods", len(methods))

	unsubscribe := func(method *url.URL) {
		log.Debugf("unsubscribe: trying to unsubscribe using %s", method.Scheme)
		var err error
		switch strings.ToLower(method.Scheme) {
		case "mailto":
			err = unsubscribeMailto(aerc, method)
		case "http", "https":
			err = unsubscribeHTTP(aerc, method)
		default:
			err = fmt.Errorf("unsubscribe: skipping unrecognized scheme: %s", method.Scheme)
		}
		if err != nil {
			aerc.PushError(err.Error())
		}
	}

	var title string = "Select method to unsubscribe"
	if msg != nil && msg.Envelope != nil && len(msg.Envelope.From) > 0 {
		title = fmt.Sprintf("%s from %s", title, msg.Envelope.From[0])
	}

	options := make([]string, len(methods))
	for i, method := range methods {
		options[i] = method.Scheme
	}

	dialog := widgets.NewSelectorDialog(
		title,
		"Press <Enter> to confirm or <ESC> to cancel",
		options, 0, aerc.SelectedAccountUiConfig(),
		func(option string, err error) {
			aerc.CloseDialog()
			if err != nil {
				if errors.Is(err, widgets.ErrNoOptionSelected) {
					aerc.PushStatus("Unsubscribe: "+err.Error(),
						5*time.Second)
				} else {
					aerc.PushError("Unsubscribe: " + err.Error())
				}
				return
			}
			for _, m := range methods {
				if m.Scheme == option {
					unsubscribe(m)
					return
				}
			}
			aerc.PushError("Unsubscribe: selected method not found")
		},
	)
	aerc.AddDialog(dialog)

	return nil
}

// parseUnsubscribeMethods reads the list-unsubscribe header and parses it as a
// list of angle-bracket <> deliminated URLs. See RFC 2369.
func parseUnsubscribeMethods(header string) (methods []*url.URL) {
	r := bufio.NewReader(strings.NewReader(header))
	for {
		// discard until <
		_, err := r.ReadSlice('<')
		if err != nil {
			return
		}
		// read until <
		m, err := r.ReadSlice('>')
		if err != nil {
			return
		}
		m = m[:len(m)-1]
		if u, err := url.Parse(string(m)); err == nil {
			methods = append(methods, u)
		}
	}
}

func unsubscribeMailto(aerc *widgets.Aerc, u *url.URL) error {
	widget := aerc.SelectedTabContent().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	h := &mail.Header{}
	h.SetSubject(u.Query().Get("subject"))
	if to, err := mail.ParseAddressList(u.Opaque); err == nil {
		h.SetAddressList("to", to)
	}

	composer, err := widgets.NewComposer(
		aerc,
		acct,
		acct.AccountConfig(),
		acct.Worker(),
		"",
		h,
		models.OriginalMail{},
	)
	if err != nil {
		return err
	}
	composer.SetContents(strings.NewReader(u.Query().Get("body")))
	tab := aerc.NewTab(composer, "unsubscribe")
	composer.OnHeaderChange("Subject", func(subject string) {
		if subject == "" {
			tab.Name = "unsubscribe"
		} else {
			tab.Name = subject
		}
		ui.Invalidate()
	})
	composer.FocusTerminal()
	return nil
}

func unsubscribeHTTP(aerc *widgets.Aerc, u *url.URL) error {
	confirm := widgets.NewSelectorDialog(
		"Do you want to open this link?",
		u.String(),
		[]string{"No", "Yes"}, 0, aerc.SelectedAccountUiConfig(),
		func(option string, _ error) {
			aerc.CloseDialog()
			switch option {
			case "Yes":
				go func() {
					if err := lib.XDGOpen(u.String()); err != nil {
						aerc.PushError("Unsubscribe:" + err.Error())
					}
				}()
			default:
				aerc.PushError("Unsubscribe: link will not be opened")
			}
		},
	)
	aerc.AddDialog(confirm)
	return nil
}
