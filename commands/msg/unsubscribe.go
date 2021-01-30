package msg

import (
	"bufio"
	"errors"
	"net/url"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
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
	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	headers := msg.RFC822Headers
	if !headers.Has("list-unsubscribe") {
		return errors.New("No List-Unsubscribe header found")
	}
	methods := parseUnsubscribeMethods(headers.Get("list-unsubscribe"))
	aerc.Logger().Printf("found %d unsubscribe methods", len(methods))
	for _, method := range methods {
		aerc.Logger().Printf("trying to unsubscribe using %v", method)
		switch method.Scheme {
		case "mailto":
			return unsubscribeMailto(aerc, method)
		case "http", "https":
			return unsubscribeHTTP(method)
		default:
			aerc.Logger().Printf("skipping unrecognized scheme: %s", method.Scheme)
		}
	}
	return errors.New("no supported unsubscribe methods found")
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
	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()

	h := &mail.Header{}
	h.SetSubject(u.Query().Get("subject"))
	if to, err := mail.ParseAddressList(u.Opaque); err == nil {
		h.SetAddressList("to", to)
	}

	composer, err := widgets.NewComposer(
		aerc,
		acct,
		aerc.Config(),
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
		tab.Content.Invalidate()
	})
	return nil
}

func unsubscribeHTTP(u *url.URL) error {
	return lib.NewXDGOpen(u.String()).Start()
}
