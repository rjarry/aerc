package authres

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-msgauth/authres"
)

const (
	AuthHeader = "Authentication-Results"
)

type Method string

const (
	DKIM  Method = "dkim"
	SPF   Method = "spf"
	DMARC Method = "dmarc"
)

type Result string

const (
	ResultNone    Result = "none"
	ResultPass    Result = "pass"
	ResultFail    Result = "fail"
	ResultNeutral Result = "neutral"
	ResultPolicy  Result = "policy"
)

type Details struct {
	Results []Result
	Infos   []string
	Reasons []string
	Err     error
}

func (d *Details) add(r Result, info string, reason string) {
	d.Results = append(d.Results, r)
	d.Infos = append(d.Infos, info)
	d.Reasons = append(d.Reasons, reason)
}

type ParserFunc func(*mail.Header, []string) (*Details, error)

func New(s string) ParserFunc {
	if i := strings.IndexRune(s, '+'); i > 0 {
		s = s[:i]
	}
	m := Method(strings.ToLower(s))
	switch m {
	case DKIM, SPF, DMARC:
		return CreateParser(m)
	}
	return nil
}

func trust(s string, trusted []string) bool {
	for _, t := range trusted {
		if matched, _ := regexp.MatchString(t, s); matched || t == "*" {
			return true
		}
	}
	return false
}

var cleaner = regexp.MustCompile(`(\(.*);(.*\))`)

func CreateParser(m Method) func(*mail.Header, []string) (*Details, error) {
	return func(header *mail.Header, trusted []string) (*Details, error) {
		details := &Details{}
		found := false

		hf := header.FieldsByKey(AuthHeader)
		for hf.Next() {
			headerText, err := hf.Text()
			if err != nil {
				return nil, err
			}

			identifier, results, err := authres.Parse(headerText)
			// TODO: refactor to use errors.Is
			switch {
			case err != nil && err.Error() == "msgauth: unsupported version":
				// Some MTA write their authres header without an identifier
				// which does not conform to RFC but still exists in the wild
				identifier, results, err = authres.Parse("unknown;" + headerText)
				if err != nil {
					return nil, err
				}
			case err != nil && err.Error() == "msgauth: malformed authentication method and value":
				// the go-msgauth parser doesn't like semi-colons in the comments
				// as a work-around we remove those
				cleanHeader := cleaner.ReplaceAllString(headerText, "${1}${2}")
				identifier, results, err = authres.Parse(cleanHeader)
				if err != nil {
					return nil, err
				}
			case err != nil:
				return nil, err
			}

			// implements recommendation from RFC 7601 Sec 7.1 to
			// have an explicit list of trustworthy hostnames
			// before displaying AuthRes results
			if !trust(identifier, trusted) {
				return nil, fmt.Errorf("%s is not trusted", identifier)
			}

			for _, result := range results {
				switch r := result.(type) {
				case *authres.DKIMResult:
					if m == DKIM {
						info := r.Identifier
						if info == "" && r.Domain != "" {
							info = r.Domain
						}
						details.add(Result(r.Value), info, r.Reason)
						found = true
					}
				case *authres.SPFResult:
					if m == SPF {
						info := r.From
						if info == "" && r.Helo != "" {
							info = r.Helo
						}
						details.add(Result(r.Value), info, r.Reason)
						found = true
					}
				case *authres.DMARCResult:
					if m == DMARC {
						details.add(Result(r.Value), r.From, r.Reason)
						found = true
					}
				}
			}
		}

		if !found {
			details.add(ResultNone, "", "")
		}
		return details, nil
	}
}
