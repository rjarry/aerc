package config

import (
	"regexp"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

type ComposeConfig struct {
	Editor              string         `ini:"editor"`
	HeaderLayout        [][]string     `ini:"header-layout" parse:"ParseLayout" default:"To|From,Subject"`
	AddressBookCmd      string         `ini:"address-book-cmd"`
	ReplyToSelf         bool           `ini:"reply-to-self" default:"true"`
	NoAttachmentWarning *regexp.Regexp `ini:"no-attachment-warning" parse:"ParseNoAttachmentWarning"`
	EmptySubjectWarning bool           `ini:"empty-subject-warning"`
	FilePickerCmd       string         `ini:"file-picker-cmd"`
	FormatFlowed        bool           `ini:"format-flowed"`
	EditHeaders         bool           `ini:"edit-headers"`
	FocusBody           bool           `ini:"focus-body"`
	LFEditor            bool           `ini:"lf-editor"`
}

var composeConfig atomic.Pointer[ComposeConfig]

func Compose() *ComposeConfig {
	return composeConfig.Load()
}

func parseCompose(file *ini.File) (*ComposeConfig, error) {
	conf := new(ComposeConfig)
	if err := MapToStruct(file.Section("compose"), conf, true); err != nil {
		return nil, err
	}
	log.Debugf("aerc.conf: [compose] %#v", conf)
	return conf, nil
}

func (c *ComposeConfig) ParseLayout(sec *ini.Section, key *ini.Key) ([][]string, error) {
	layout := parseLayout(key.String())
	return layout, nil
}

func (c *ComposeConfig) ParseNoAttachmentWarning(sec *ini.Section, key *ini.Key) (*regexp.Regexp, error) {
	if key.String() == "" {
		return nil, nil
	}
	return regexp.Compile(`(?im)` + key.String())
}
