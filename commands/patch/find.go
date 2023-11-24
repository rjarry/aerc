package patch

import (
	"errors"
	"fmt"
	"net/textproto"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/go-opt"
)

type Find struct {
	Filter bool     `opt:"-f"`
	Commit []string `opt:"..." required:"true" complete:"Complete"`
}

func init() {
	register(Find{})
}

func (Find) Aliases() []string {
	return []string{"find"}
}

func (*Find) Complete(arg string) []string {
	m := pama.New()
	p, err := m.CurrentProject()
	if err != nil {
		return nil
	}

	options := make([]string, len(p.Commits))
	for i, c := range p.Commits {
		options[i] = fmt.Sprintf("%-6.6s %s", c.ID, c.Subject)
	}

	return commands.FilterList(options, arg, nil)
}

func (s Find) Execute(_ []string) error {
	m := pama.New()
	p, err := m.CurrentProject()
	if err != nil {
		return err
	}

	if len(s.Commit) == 0 {
		return errors.New("missing commit hash")
	}

	lexed := opt.LexArgs(strings.TrimSpace(s.Commit[0]))

	hash, err := lexed.ArgSafe(0)
	if err != nil {
		return err
	}

	if len(hash) < 4 {
		return errors.New("Commit hash is too short.")
	}

	var c models.Commit
	for _, commit := range p.Commits {
		if strings.Contains(commit.ID, hash) {
			c = commit
			break
		}
	}
	if c.ID == "" {
		var err error
		c, err = m.Find(hash, p)
		if err != nil {
			return err
		}
	}

	// If Message-Id is provided, find it in store
	if c.MessageId != "" {
		if selectMessageId(c.MessageId) {
			return nil
		}
	}

	// Fallback to a search based on the subject line
	args := []string{"search"}
	if s.Filter {
		args[0] = "filter"
	}

	headers := make(textproto.MIMEHeader)
	args = append(args, fmt.Sprintf("-H Subject:%s", c.Subject))
	headers.Add("Subject", c.Subject)

	cmd := account.SearchFilter{
		Headers: headers,
	}

	return cmd.Execute(args)
}

func selectMessageId(msgid string) bool {
	acct := app.SelectedAccount()
	if acct == nil {
		return false
	}
	store := acct.Store()
	if store == nil {
		return false
	}
	for uid, msg := range store.Messages {
		if msg == nil {
			continue
		}
		if msg.RFC822Headers == nil {
			continue
		}
		id, err := msg.RFC822Headers.MessageID()
		if err != nil {
			continue
		}
		if id == msgid {
			store.Select(uid)
			return true
		}
	}
	return false
}
