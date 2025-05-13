package format

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-message/mail"
)

const rfc5322specials string = `()<>[]:;@\,."`

// AddressForHumans formats the address.
// Meant for display purposes to the humans, not for sending over the wire.
func AddressForHumans(a *mail.Address) string {
	if a.Name != "" {
		if strings.ContainsAny(a.Name, rfc5322specials) {
			return fmt.Sprintf("\"%s\" <%s>",
				strings.ReplaceAll(a.Name, "\"", "'"), a.Address)
		} else {
			return fmt.Sprintf("%s <%s>", a.Name, a.Address)
		}
	} else {
		return fmt.Sprintf("<%s>", a.Address)
	}
}

// FormatAddresses formats a list of addresses into a human readable string
func FormatAddresses(l []*mail.Address) string {
	formatted := make([]string, len(l))
	for i, a := range l {
		formatted[i] = strings.TrimSpace(AddressForHumans(a))
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
