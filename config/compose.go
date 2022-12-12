package config

import (
	"fmt"
	"regexp"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

type ComposeConfig struct {
	Editor              string         `ini:"editor"`
	HeaderLayout        [][]string     `ini:"-"`
	AddressBookCmd      string         `ini:"address-book-cmd"`
	ReplyToSelf         bool           `ini:"reply-to-self"`
	NoAttachmentWarning *regexp.Regexp `ini:"-"`
	FilePickerCmd       string         `ini:"file-picker-cmd"`
}

func defaultComposeConfig() *ComposeConfig {
	return &ComposeConfig{
		HeaderLayout: [][]string{
			{"To", "From"},
			{"Subject"},
		},
		ReplyToSelf: true,
	}
}

var Compose = defaultComposeConfig()

func parseCompose(file *ini.File) error {
	compose, err := file.GetSection("compose")
	if err != nil {
		goto end
	}

	if err := compose.MapTo(&Compose); err != nil {
		return err
	}
	for key, val := range compose.KeysHash() {
		if key == "header-layout" {
			Compose.HeaderLayout = parseLayout(val)
		}

		if key == "no-attachment-warning" && len(val) > 0 {
			re, err := regexp.Compile("(?im)" + val)
			if err != nil {
				return fmt.Errorf(
					"Invalid no-attachment-warning '%s': %w",
					val, err,
				)
			}

			Compose.NoAttachmentWarning = re
		}
	}

end:
	log.Debugf("aerc.conf: [compose] %#v", Compose)
	return nil
}
