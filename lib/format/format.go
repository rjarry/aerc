package format

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
)

// AddressForHumans formats the address. If the address's name
// contains non-ASCII characters it will be quoted but not encoded.
// Meant for display purposes to the humans, not for sending over the wire.
func AddressForHumans(a *mail.Address) string {
	if a.Name != "" {
		if atom.MatchString(a.Name) {
			return fmt.Sprintf("%s <%s>", a.Name, a.Address)
		} else {
			return fmt.Sprintf("\"%s\" <%s>",
				strings.ReplaceAll(a.Name, "\"", "'"), a.Address)
		}
	} else {
		return fmt.Sprintf("<%s>", a.Address)
	}
}

var atom *regexp.Regexp = regexp.MustCompile("^[a-z0-9!#$%7'*+-/=?^_`{}|~ ]+$")

// FormatAddresses formats a list of addresses into a human readable string
func FormatAddresses(l []*mail.Address) string {
	formatted := make([]string, len(l))
	for i, a := range l {
		formatted[i] = AddressForHumans(a)
	}
	return strings.Join(formatted, ", ")
}

type Ctx struct {
	FromAddress string
	AccountName string
	MsgNum      int
	MsgInfo     *models.MessageInfo
	MsgIsMarked bool
}

func ParseMessageFormat(format string, timeFmt string, thisDayTimeFmt string,
	thisYearTimeFmt string, ctx Ctx) (string, []interface{}, error) {
	retval := make([]byte, 0, len(format))
	var args []interface{}

	accountFromAddress, err := mail.ParseAddress(ctx.FromAddress)
	if err != nil {
		return "", nil, err
	}

	envelope := ctx.MsgInfo.Envelope
	if envelope == nil {
		return "", nil,
			errors.New("no envelope available for this message")
	}

	var c rune
	for i, ni := 0, 0; i < len(format); {
		ni = strings.IndexByte(format[i:], '%')
		if ni < 0 {
			ni = len(format)
			retval = append(retval, []byte(format[i:ni])...)
			break
		}
		ni += i + 1
		// Check for fmt flags
		if ni == len(format) {
			goto handle_end_error
		}
		c = rune(format[ni])
		if c == '+' || c == '-' || c == '#' || c == ' ' || c == '0' {
			ni++
		}

		// Check for precision and width
		if ni == len(format) {
			goto handle_end_error
		}
		c = rune(format[ni])
		for unicode.IsDigit(c) {
			ni++
			c = rune(format[ni])
		}
		if c == '.' {
			ni++
			c = rune(format[ni])
			for unicode.IsDigit(c) {
				ni++
				c = rune(format[ni])
			}
		}

		retval = append(retval, []byte(format[i:ni])...)
		// Get final format verb
		if ni == len(format) {
			goto handle_end_error
		}
		c = rune(format[ni])
		switch c {
		case '%':
			retval = append(retval, '%')
		case 'a':
			if len(envelope.From) == 0 {
				return "", nil,
					errors.New("found no address for sender")
			}
			addr := envelope.From[0]
			retval = append(retval, 's')
			args = append(args, addr.Address)
		case 'A':
			var addr *mail.Address
			if len(envelope.ReplyTo) == 0 {
				if len(envelope.From) == 0 {
					return "", nil,
						errors.New("found no address for sender or reply-to")
				} else {
					addr = envelope.From[0]
				}
			} else {
				addr = envelope.ReplyTo[0]
			}
			retval = append(retval, 's')
			args = append(args, addr.Address)
		case 'C':
			retval = append(retval, 'd')
			args = append(args, ctx.MsgNum)
		case 'd':
			date := envelope.Date
			if date.IsZero() {
				date = ctx.MsgInfo.InternalDate
			}
			retval = append(retval, 's')
			args = append(args,
				dummyIfZeroDate(date.Local(),
					timeFmt, thisDayTimeFmt, thisYearTimeFmt))
		case 'D':
			date := envelope.Date
			if date.IsZero() {
				date = ctx.MsgInfo.InternalDate
			}
			retval = append(retval, 's')
			args = append(args,
				dummyIfZeroDate(date.Local(),
					timeFmt, thisDayTimeFmt, thisYearTimeFmt))
		case 'f':
			if len(envelope.From) == 0 {
				return "", nil,
					errors.New("found no address for sender")
			}
			addr := AddressForHumans(envelope.From[0])
			retval = append(retval, 's')
			args = append(args, addr)
		case 'F':
			if len(envelope.From) == 0 {
				return "", nil,
					errors.New("found no address for sender")
			}
			addr := envelope.From[0]
			var val string

			if addr.Name == accountFromAddress.Name && len(envelope.To) != 0 {
				addr = envelope.To[0]
			}

			if addr.Name != "" {
				val = addr.Name
			} else {
				val = addr.Address
			}
			retval = append(retval, 's')
			args = append(args, val)

		case 'g':
			retval = append(retval, 's')
			args = append(args, strings.Join(ctx.MsgInfo.Labels, ", "))

		case 'i':
			retval = append(retval, 's')
			args = append(args, envelope.MessageId)
		case 'n':
			if len(envelope.From) == 0 {
				return "", nil,
					errors.New("found no address for sender")
			}
			addr := envelope.From[0]
			var val string
			if addr.Name != "" {
				val = addr.Name
			} else {
				val = addr.Address
			}
			retval = append(retval, 's')
			args = append(args, val)
		case 'r':
			addrs := FormatAddresses(envelope.To)
			retval = append(retval, 's')
			args = append(args, addrs)
		case 'R':
			addrs := FormatAddresses(envelope.Cc)
			retval = append(retval, 's')
			args = append(args, addrs)
		case 's':
			retval = append(retval, 's')
			args = append(args, envelope.Subject)
		case 't':
			if len(envelope.To) == 0 {
				return "", nil,
					errors.New("found no address for recipient")
			}
			addr := envelope.To[0]
			retval = append(retval, 's')
			args = append(args, addr.Address)
		case 'T':
			retval = append(retval, 's')
			args = append(args, ctx.AccountName)
		case 'u':
			if len(envelope.From) == 0 {
				return "", nil,
					errors.New("found no address for sender")
			}
			addr := envelope.From[0]
			mailbox := addr.Address // fallback if there's no @ sign
			if split := strings.SplitN(addr.Address, "@", 2); len(split) == 2 {
				mailbox = split[1]
			}
			retval = append(retval, 's')
			args = append(args, mailbox)
		case 'v':
			if len(envelope.From) == 0 {
				return "", nil,
					errors.New("found no address for sender")
			}
			addr := envelope.From[0]
			// check if message is from current user
			if addr.Name != "" {
				retval = append(retval, 's')
				args = append(args,
					strings.Split(addr.Name, " ")[0])
			}
		case 'Z':
			// calculate all flags
			var readReplyFlag = ""
			var delFlag = ""
			var flaggedFlag = ""
			var markedFlag = ""
			seen := false
			recent := false
			answered := false
			for _, flag := range ctx.MsgInfo.Flags {
				if flag == models.SeenFlag {
					seen = true
				} else if flag == models.RecentFlag {
					recent = true
				} else if flag == models.AnsweredFlag {
					answered = true
				}
				if flag == models.DeletedFlag {
					delFlag = "D"
					// TODO: check if attachments
				}
				if flag == models.FlaggedFlag {
					flaggedFlag = "!"
				}
				// TODO: check gpg stuff
			}
			if seen {
				if answered {
					readReplyFlag = "r" // message has been replied to
				}
			} else {
				if recent {
					readReplyFlag = "N" // message is new
				} else {
					readReplyFlag = "O" // message is old
				}
			}
			if ctx.MsgIsMarked {
				markedFlag = "*"
			}
			retval = append(retval, '4', 's')
			args = append(args, readReplyFlag+delFlag+flaggedFlag+markedFlag)

		// Move the below cases to proper alphabetical positions once
		// implemented
		case 'l':
			// TODO: number of lines in the message
			retval = append(retval, 'd')
			args = append(args, ctx.MsgInfo.Size)
		case 'e':
			// TODO: current message number in thread
			fallthrough
		case 'E':
			// TODO: number of messages in current thread
			fallthrough
		case 'H':
			// TODO: spam attribute(s) of this message
			fallthrough
		case 'L':
			// TODO:
			fallthrough
		case 'X':
			// TODO: number of attachments
			fallthrough
		case 'y':
			// TODO: X-Label field
			fallthrough
		case 'Y':
			// TODO: X-Label field and some other constraints
			fallthrough
		default:
			// Just ignore it and print as is
			// so %k in index format becomes %%k to Printf
			retval = append(retval, '%')
			retval = append(retval, byte(c))
		}
		i = ni + 1
	}

	return string(retval), args, nil

handle_end_error:
	return "", nil,
		errors.New("reached end of string while parsing message format")
}

func dummyIfZeroDate(date time.Time, format string, todayFormat string,
	thisYearFormat string) string {
	if date.IsZero() {
		return strings.Repeat("?", len(format))
	}
	year, month, day := date.Date()
	thisYear, thisMonth, thisDay := time.Now().Date()
	if year == thisYear {
		if month == thisMonth && day == thisDay && todayFormat != "" {
			return date.Format(todayFormat)
		}
		if thisYearFormat != "" {
			return date.Format(thisYearFormat)
		}
	}
	return date.Format(format)
}
