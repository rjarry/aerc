package config

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"github.com/go-ini/ini"
)

type ColumnFlags uint32

func (f ColumnFlags) Has(o ColumnFlags) bool { return f&o == o }

const (
	ALIGN_LEFT ColumnFlags = 1 << iota
	ALIGN_CENTER
	ALIGN_RIGHT
	WIDTH_AUTO     // whatever is left
	WIDTH_FRACTION // ratio of total width
	WIDTH_EXACT    // exact number of characters
	WIDTH_FIT      // fit to column content width
)

type ColumnDef struct {
	Name     string
	Flags    ColumnFlags
	Width    float64
	Template *template.Template
}

var columnRe = regexp.MustCompile(`^([\w-]+)(?:([<:>])(=|\*|\d+%?)?)?$`)

func parseColumnDef(col string, section *ini.Section) (*ColumnDef, error) {
	col = strings.TrimSpace(col)
	match := columnRe.FindStringSubmatch(col)
	if match == nil {
		return nil, fmt.Errorf("invalid column def: %v", col)
	}
	name := match[1]
	keyName := fmt.Sprintf("column-%s", name)

	var flags ColumnFlags
	switch match[2] {
	case "<", "":
		flags |= ALIGN_LEFT
	case ":":
		flags |= ALIGN_CENTER
	case ">":
		flags |= ALIGN_RIGHT
	}

	var width float64 = 0
	switch match[3] {
	case "=":
		flags |= WIDTH_FIT
	case "*", "":
		flags |= WIDTH_AUTO
	default:
		s := match[3]
		var divider float64 = 1
		if strings.HasSuffix(s, "%") {
			divider = 100
			s = strings.TrimSuffix(s, "%")
			flags |= WIDTH_FRACTION
		} else {
			flags |= WIDTH_EXACT
		}
		w, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", keyName, err)
		}
		if divider == 100 && w > 100 {
			return nil, fmt.Errorf("%s: invalid width %.0f%%", keyName, w)
		}
		width = w / divider
	}
	key, err := section.GetKey(keyName)
	if err != nil {
		return nil, err
	}

	t, err := templates.ParseTemplate(keyName, key.String())
	if err != nil {
		return nil, err
	}

	err = templates.Render(t, &bytes.Buffer{}, &dummyData{})
	if err != nil {
		return nil, err
	}

	return &ColumnDef{
		Name:     name,
		Flags:    flags,
		Width:    width,
		Template: t,
	}, nil
}

func ParseColumnDefs(key *ini.Key, section *ini.Section) ([]*ColumnDef, error) {
	var columns []*ColumnDef
	for _, col := range key.Strings(",") {
		c, err := parseColumnDef(col, section)
		if err != nil {
			return nil, err
		}
		columns = append(columns, c)
	}
	if len(columns) == 0 {
		return nil, nil
	}
	return columns, nil
}
