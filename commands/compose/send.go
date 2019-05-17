package compose

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/gdamore/tcell"
	"github.com/miolini/datacounter"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func init() {
	register("send", SendMessage)
}

func SendMessage(aerc *widgets.Aerc, args []string) error {
	if len(args) > 1 {
		return errors.New("Usage: send-message")
	}
	composer, _ := aerc.SelectedTab().(*widgets.Composer)
	config := composer.Config()

	if config.Outgoing == "" {
		return errors.New(
			"No outgoing mail transport configured for this account")
	}

	uri, err := url.Parse(config.Outgoing)
	if err != nil {
		return err
	}
	var (
		scheme string
		auth   string = "plain"
	)
	parts := strings.Split(uri.Scheme, "+")
	if len(parts) == 1 {
		scheme = parts[0]
	} else if len(parts) == 2 {
		scheme = parts[0]
		auth = parts[1]
	} else {
		return fmt.Errorf("Unknown transfer protocol %s", uri.Scheme)
	}

	header, rcpts, err := composer.PrepareHeader()
	if err != nil {
		return err
	}

	if config.From == "" {
		return errors.New("No 'From' configured for this account")
	}
	from, err := mail.ParseAddress(config.From)
	if err != nil {
		return err
	}

	var (
		saslClient sasl.Client
		conn       *smtp.Client
	)
	switch auth {
	case "":
		fallthrough
	case "none":
		saslClient = nil
	case "plain":
		password, _ := uri.User.Password()
		saslClient = sasl.NewPlainClient("", uri.User.Username(), password)
	default:
		return fmt.Errorf("Unsupported auth mechanism %s", auth)
	}

	aerc.RemoveTab(composer)

	sendAsync := func() (int, error) {
		tlsConfig := &tls.Config{
			// TODO: ask user first
			InsecureSkipVerify: true,
		}
		switch scheme {
		case "smtp":
			host := uri.Host
			if !strings.ContainsRune(host, ':') {
				host = host + ":587" // Default to submission port
			}
			conn, err = smtp.Dial(host)
			if err != nil {
				aerc.PushStatus(" "+err.Error(), 10*time.Second).
					Color(tcell.ColorDefault, tcell.ColorRed)
				return 0, nil
			}
			defer conn.Close()
			if sup, _ := conn.Extension("STARTTLS"); sup {
				// TODO: let user configure tls?
				if err = conn.StartTLS(tlsConfig); err != nil {
					aerc.PushStatus(" "+err.Error(), 10*time.Second).
						Color(tcell.ColorDefault, tcell.ColorRed)
					return 0, nil
				}
			}
		case "smtps":
			host := uri.Host
			if !strings.ContainsRune(host, ':') {
				host = host + ":465" // Default to smtps port
			}
			conn, err = smtp.DialTLS(host, tlsConfig)
			if err != nil {
				aerc.PushStatus(" "+err.Error(), 10*time.Second).
					Color(tcell.ColorDefault, tcell.ColorRed)
				return 0, nil
			}
			defer conn.Close()
		}

		// TODO: sendmail
		if saslClient != nil {
			if err = conn.Auth(saslClient); err != nil {
				aerc.PushStatus(" "+err.Error(), 10*time.Second).
					Color(tcell.ColorDefault, tcell.ColorRed)
				return 0, nil
			}
		}
		// TODO: the user could conceivably want to use a different From and sender
		if err = conn.Mail(from.Address); err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
			return 0, nil
		}
		for _, rcpt := range rcpts {
			if err = conn.Rcpt(rcpt); err != nil {
				aerc.PushStatus(" "+err.Error(), 10*time.Second).
					Color(tcell.ColorDefault, tcell.ColorRed)
				return 0, nil
			}
		}
		wc, err := conn.Data()
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
			return 0, nil
		}
		defer wc.Close()
		ctr := datacounter.NewWriterCounter(wc)
		composer.WriteMessage(header, ctr)
		return int(ctr.Count()), nil
	}

	go func() {
		aerc.SetStatus("Sending...")
		nbytes, err := sendAsync()
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
			return
		}
		if config.CopyTo != "" {
			aerc.SetStatus("Copying to " + config.CopyTo)
			worker := composer.Worker()
			r, w := io.Pipe()
			worker.PostAction(&types.AppendMessage{
				Destination: config.CopyTo,
				Flags: []string{},
				Date: time.Now(),
				Reader: r,
				Length: nbytes,
			}, func(msg types.WorkerMessage) {
				switch msg := msg.(type) {
				case *types.Done:
					aerc.SetStatus("Message sent.")
					r.Close()
					composer.Close()
				case *types.Error:
					aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
						Color(tcell.ColorDefault, tcell.ColorRed)
					r.Close()
					composer.Close()
				}
			})
			header, _, _ := composer.PrepareHeader()
			composer.WriteMessage(header, w)
			w.Close()
		} else {
			aerc.SetStatus("Message sent.")
			composer.Close()
		}
	}()
	return nil
}
