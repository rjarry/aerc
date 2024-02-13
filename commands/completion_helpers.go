package commands

import (
	"fmt"
	"net/mail"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/completer"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
)

// GetAddress uses the address-book-cmd for address completion
func GetAddress(search string) []string {
	var options []string

	cmd := app.SelectedAccount().AccountConfig().AddressBookCmd
	if cmd == "" {
		cmd = config.Compose.AddressBookCmd
		if cmd == "" {
			return nil
		}
	}

	cmpl := completer.New(cmd, func(err error) {
		app.PushError(
			fmt.Sprintf("could not complete header: %v", err))
		log.Warnf("could not complete header: %v", err)
	})

	if len(search) > config.Ui.CompletionMinChars && cmpl != nil {
		addrList, _ := cmpl.ForHeader("to")(search)
		for _, full := range addrList {
			addr, err := mail.ParseAddress(full)
			if err != nil {
				continue
			}
			options = append(options, addr.Address)
		}
	}

	return options
}

// GetFlagList returns a list of available flags for completion
func GetFlagList() []string {
	return []string{"Seen", "Answered", "Flagged", "Draft"}
}

// GetDateList returns a list of date terms for completion
func GetDateList() []string {
	return []string{
		"today", "yesterday", "this_week", "this_month",
		"this_year", "last_week", "last_month", "last_year",
		"Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
		"Saturday", "Sunday",
	}
}

// Operands returns a slice without any option flags or mandatory option
// arguments
func Operands(args []string, spec string) []string {
	var result []string
	for i := 0; i < len(args); i++ {
		if s := args[i]; s == "--" {
			return args[i+1:]
		} else if strings.HasPrefix(s, "-") && len(spec) > 0 {
			r := string(s[len(s)-1]) + ":"
			if strings.Contains(spec, r) {
				i++
			}
			continue
		}
		result = append(result, args[i])
	}
	return result
}
