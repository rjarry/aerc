package format

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-message/mail"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
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

// CompactPath reduces a directory path into a compact form.  The directory
// name will be split with the provided separator and each part will be reduced
// to the first letter in its name: INBOX/01_WORK/PROJECT  will become
// I/W/PROJECT.
func CompactPath(name string, sep rune) (compact string) {
	parts := strings.Split(name, string(sep))
	for i, part := range parts {
		if i == len(parts)-1 {
			compact += part
		} else {
			if len(part) != 0 {
				r := part[0]
				for i := 0; i < len(part)-1; i++ {
					if unicode.IsLetter(rune(part[i])) {
						r = part[i]
						break
					}
				}
				compact += fmt.Sprintf("%c%c", r, sep)
			} else {
				compact += fmt.Sprintf("%c", sep)
			}
		}
	}
	return
}

func TruncateHead(s string, w int, head string) string {
	width := runewidth.StringWidth(s)
	if width <= w {
		return s
	}
	w -= runewidth.StringWidth(head)
	pos := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		var chWidth int
		for _, r := range g.Runes() {
			chWidth = runewidth.RuneWidth(r)
			if chWidth > 0 {
				break
			}
		}
		if width-chWidth <= w {
			pos, _ = g.Positions()
			break
		}
		width -= chWidth
	}
	return head + s[pos:]
}

func DummyIfZeroDate(date time.Time, format string, todayFormat string,
	thisWeekFormat string, thisYearFormat string,
) string {
	if date.IsZero() {
		return strings.Repeat("?", len(format))
	}
	year := date.Year()
	day := date.YearDay()
	now := time.Now()
	thisYear := now.Year()
	thisDay := now.YearDay()
	if year == thisYear {
		if day == thisDay && todayFormat != "" {
			return date.Format(todayFormat)
		}
		if day > thisDay-7 && thisWeekFormat != "" {
			return date.Format(thisWeekFormat)
		}
		if thisYearFormat != "" {
			return date.Format(thisYearFormat)
		}
	}
	return date.Format(format)
}
