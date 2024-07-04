package account

import (
	"errors"
	"fmt"
	"net/textproto"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/imap/extensions/xgmext"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type SearchFilter struct {
	Read         bool                 `opt:"-r" action:"ParseRead"`
	Unread       bool                 `opt:"-u" action:"ParseUnread"`
	Body         bool                 `opt:"-b"`
	All          bool                 `opt:"-a"`
	UseExtension bool                 `opt:"-e"`
	Headers      textproto.MIMEHeader `opt:"-H" action:"ParseHeader" metavar:"<header>:<value>"`
	WithFlags    models.Flags         `opt:"-x" action:"ParseFlag" complete:"CompleteFlag"`
	WithoutFlags models.Flags         `opt:"-X" action:"ParseNotFlag" complete:"CompleteFlag"`
	To           []string             `opt:"-t" action:"ParseTo" complete:"CompleteAddress"`
	From         []string             `opt:"-f" action:"ParseFrom" complete:"CompleteAddress"`
	Cc           []string             `opt:"-c" action:"ParseCc" complete:"CompleteAddress"`
	StartDate    time.Time            `opt:"-d" action:"ParseDate" complete:"CompleteDate"`
	EndDate      time.Time
	Terms        string `opt:"..." required:"false" complete:"CompleteTerms"`
}

func init() {
	commands.Register(SearchFilter{})
}

func (SearchFilter) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (SearchFilter) Aliases() []string {
	return []string{"search", "filter"}
}

func (*SearchFilter) CompleteFlag(arg string) []string {
	return commands.FilterList(commands.GetFlagList(), arg, commands.QuoteSpace)
}

func (*SearchFilter) CompleteAddress(arg string) []string {
	return commands.FilterList(commands.GetAddress(arg), arg, commands.QuoteSpace)
}

func (*SearchFilter) CompleteDate(arg string) []string {
	return commands.FilterList(commands.GetDateList(), arg, commands.QuoteSpace)
}

func (s *SearchFilter) CompleteTerms(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}
	if acct.AccountConfig().Backend == "notmuch" {
		return handleNotmuchComplete(arg)
	}
	caps := acct.Worker().Backend.Capabilities()
	if caps != nil && caps.Has("X-GM-EXT-1") && s.UseExtension {
		return handleXGMEXTComplete(arg)
	}
	return nil
}

func (s *SearchFilter) ParseRead(arg string) error {
	s.WithFlags |= models.SeenFlag
	s.WithoutFlags &^= models.SeenFlag
	return nil
}

func (s *SearchFilter) ParseUnread(arg string) error {
	s.WithFlags &^= models.SeenFlag
	s.WithoutFlags |= models.SeenFlag
	return nil
}

var flagValues = map[string]models.Flags{
	"seen":     models.SeenFlag,
	"answered": models.AnsweredFlag,
	"flagged":  models.FlaggedFlag,
	"draft":    models.DraftFlag,
}

func (s *SearchFilter) ParseFlag(arg string) error {
	f, ok := flagValues[strings.ToLower(arg)]
	if !ok {
		return fmt.Errorf("%q unknown flag", arg)
	}
	s.WithFlags |= f
	s.WithoutFlags &^= f
	return nil
}

func (s *SearchFilter) ParseNotFlag(arg string) error {
	f, ok := flagValues[strings.ToLower(arg)]
	if !ok {
		return fmt.Errorf("%q unknown flag", arg)
	}
	s.WithFlags &^= f
	s.WithoutFlags |= f
	return nil
}

func (s *SearchFilter) ParseHeader(arg string) error {
	name, value, hasColon := strings.Cut(arg, ":")
	if !hasColon {
		return fmt.Errorf("%q invalid syntax", arg)
	}
	if s.Headers == nil {
		s.Headers = make(textproto.MIMEHeader)
	}
	s.Headers.Add(name, strings.TrimSpace(value))
	return nil
}

func (s *SearchFilter) ParseTo(arg string) error {
	s.To = append(s.To, arg)
	return nil
}

func (s *SearchFilter) ParseFrom(arg string) error {
	s.From = append(s.From, arg)
	return nil
}

func (s *SearchFilter) ParseCc(arg string) error {
	s.Cc = append(s.Cc, arg)
	return nil
}

func (s *SearchFilter) ParseDate(arg string) error {
	start, end, err := parse.DateRange(arg)
	if err != nil {
		return err
	}
	s.StartDate = start
	s.EndDate = end
	return nil
}

func (s SearchFilter) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	criteria := types.SearchCriteria{
		WithFlags:    s.WithFlags,
		WithoutFlags: s.WithoutFlags,
		From:         s.From,
		To:           s.To,
		Cc:           s.Cc,
		Headers:      s.Headers,
		StartDate:    s.StartDate,
		EndDate:      s.EndDate,
		SearchBody:   s.Body,
		SearchAll:    s.All,
		Terms:        []string{s.Terms},
		UseExtension: s.UseExtension,
	}

	if args[0] == "filter" {
		if len(args[1:]) == 0 {
			return Clear{}.Execute([]string{"clear"})
		}
		acct.SetStatus(state.FilterActivity("Filtering..."), state.Search(""))
		store.SetFilter(&criteria)
		cb := func(msg types.WorkerMessage) {
			if _, ok := msg.(*types.Done); ok {
				acct.SetStatus(state.FilterResult(strings.Join(args, " ")))
				log.Tracef("Filter results: %v", store.Uids())
			}
		}
		store.Sort(store.GetCurrentSortCriteria(), cb)
	} else {
		acct.SetStatus(state.Search("Searching..."))
		cb := func(uids []uint32) {
			acct.SetStatus(state.Search(strings.Join(args, " ")))
			log.Tracef("Search results: %v", uids)
			store.ApplySearch(uids)
			// TODO: Remove when stores have multiple OnUpdate handlers
			ui.Invalidate()
		}
		store.Search(&criteria, cb)
	}
	return nil
}

func handleXGMEXTComplete(arg string) []string {
	prefixes := []string{"from:", "to:", "deliveredto:", "cc:", "bcc:"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(arg, prefix) {
			arg = strings.TrimPrefix(arg, prefix)
			return commands.FilterList(
				commands.GetAddress(arg), arg,
				func(v string) string { return prefix + v },
			)
		}
	}

	return commands.FilterList(xgmext.Terms, arg, nil)
}
