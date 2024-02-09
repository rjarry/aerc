package config

import (
	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

type StatuslineConfig struct {
	StatusColumns   []*ColumnDef `ini:"status-columns" parse:"ParseColumns" default:"left<*,center>=,right>*"`
	ColumnSeparator string       `ini:"column-separator" default:" "`
	Separator       string       `ini:"separator" default:" | "`
	DisplayMode     string       `ini:"display-mode" default:"text"`
}

var Statusline = new(StatuslineConfig)

func parseStatusline(file *ini.File) error {
	statusline := file.Section("statusline")
	if err := MapToStruct(statusline, Statusline, true); err != nil {
		return err
	}

	log.Debugf("aerc.conf: [statusline] %#v", Statusline)
	return nil
}

func (s *StatuslineConfig) ParseColumns(sec *ini.Section, key *ini.Key) ([]*ColumnDef, error) {
	if !sec.HasKey("column-left") {
		_, _ = sec.NewKey("column-left", "[{{.Account}}] {{.StatusInfo}}")
	}
	if !sec.HasKey("column-center") {
		_, _ = sec.NewKey("column-center", "{{.PendingKeys}}")
	}
	if !sec.HasKey("column-right") {
		_, _ = sec.NewKey("column-right", "{{.TrayInfo}} | {{cwd}}")
	}
	return ParseColumnDefs(key, sec)
}
