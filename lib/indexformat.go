package lib

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func ParseIndexFormat(conf *config.AercConfig, number int,
	msg *types.MessageInfo) (string, []interface{}, error) {

	format := conf.Ui.IndexFormat
	retval := make([]byte, 0, len(format))
	var args []interface{}

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
			if len(msg.Envelope.From) == 0 {
				return "", nil, errors.New("found no address for sender")
			}
			addr := msg.Envelope.From[0]
			retval = append(retval, 's')
			args = append(args, fmt.Sprintf("%s@%s", addr.MailboxName,
				addr.HostName))
		case 'A':
			var addr *imap.Address
			if len(msg.Envelope.ReplyTo) == 0 {
				if len(msg.Envelope.From) == 0 {
					return "", nil,
						errors.New("found no address for sender or reply-to")
				} else {
					addr = msg.Envelope.From[0]
				}
			} else {
				addr = msg.Envelope.ReplyTo[0]
			}
			retval = append(retval, 's')
			args = append(args, fmt.Sprintf("%s@%s", addr.MailboxName,
				addr.HostName))
		case 'C':
			retval = append(retval, 'd')
			args = append(args, number)
		case 'd':
			retval = append(retval, 's')
			args = append(args, msg.InternalDate.Format(conf.Ui.TimestampFormat))
		case 'D':
			retval = append(retval, 's')
			args = append(args, msg.InternalDate.Local().Format(conf.Ui.TimestampFormat))
		case 'f':
			if len(msg.Envelope.From) == 0 {
				return "", nil, errors.New("found no address for sender")
			}
			addr := FormatAddress(msg.Envelope.From[0])
			retval = append(retval, 's')
			args = append(args, addr)
		case 'F':
			if len(msg.Envelope.From) == 0 {
				return "", nil, errors.New("found no address for sender")
			}
			addr := msg.Envelope.From[0]
			// TODO: handle case when sender is current user. Then
			// use recipient's name
			var val string
			if addr.PersonalName != "" {
				val = addr.PersonalName
			} else {
				val = fmt.Sprintf("%s@%s",
					addr.MailboxName, addr.HostName)
			}
			retval = append(retval, 's')
			args = append(args, val)

		case 'i':
			retval = append(retval, 's')
			args = append(args, msg.Envelope.MessageId)
		case 'n':
			if len(msg.Envelope.From) == 0 {
				return "", nil, errors.New("found no address for sender")
			}
			addr := msg.Envelope.From[0]
			var val string
			if addr.PersonalName != "" {
				val = addr.PersonalName
			} else {
				val = fmt.Sprintf("%s@%s",
					addr.MailboxName, addr.HostName)
			}
			retval = append(retval, 's')
			args = append(args, val)
		case 'r':
			addrs := FormatAddresses(msg.Envelope.To)
			retval = append(retval, 's')
			args = append(args, addrs)
		case 'R':
			addrs := FormatAddresses(msg.Envelope.Cc)
			retval = append(retval, 's')
			args = append(args, addrs)
		case 's':
			retval = append(retval, 's')
			args = append(args, msg.Envelope.Subject)
		case 'u':
			if len(msg.Envelope.From) == 0 {
				return "", nil, errors.New("found no address for sender")
			}
			addr := msg.Envelope.From[0]
			retval = append(retval, 's')
			args = append(args, addr.MailboxName)
		case 'v':
			if len(msg.Envelope.From) == 0 {
				return "", nil, errors.New("found no address for sender")
			}
			addr := msg.Envelope.From[0]
			// check if message is from current user
			if addr.PersonalName != "" {
				retval = append(retval, 's')
				args = append(args, strings.Split(addr.PersonalName, " ")[0])
			}
		case 'Z':
			// calculate all flags
			var readFlag = ""
			var delFlag = ""
			var flaggedFlag = ""
			for _, flag := range msg.Flags {
				if flag == imap.SeenFlag {
					readFlag = "O" // message is old
				} else if flag == imap.RecentFlag {
					readFlag = "N" // message is new
				} else if flag == imap.AnsweredFlag {
					readFlag = "r" // message has been replied to
				}
				if flag == imap.DeletedFlag {
					delFlag = "D"
					// TODO: check if attachments
				}
				if flag == imap.FlaggedFlag {
					flaggedFlag = "!"
				}
				// TODO: check gpg stuff
			}
			retval = append(retval, '3', 's')
			args = append(args, readFlag+delFlag+flaggedFlag)

		// Move the below cases to proper alphabetical positions once
		// implemented
		case 'l':
			// TODO: number of lines in the message
			retval = append(retval, 'd')
			args = append(args, msg.Size)
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
	return "", nil, errors.New("reached end of string while parsing index format")
}
