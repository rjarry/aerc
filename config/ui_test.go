package config

import (
	"reflect"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func templateText(t *template.Template) string {
	// unfortunately, the original template text is stored as a private
	// field, for test purposes, access its value via reflection
	v := reflect.ValueOf(t).Elem()
	return v.FieldByName("text").String()
}

func TestConvertIndexFormat(t *testing.T) {
	columns, err := convertIndexFormat("%-20.20D %-17.17n %Z %s")
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, columns, 4)

	assert.Equal(t, "date", columns[0].Name)
	assert.Equal(t, 20.0, columns[0].Width)
	assert.Equal(t, ALIGN_LEFT|WIDTH_EXACT, columns[0].Flags)
	assert.Equal(t, `{{.DateAutoFormat .Date.Local}}`,
		templateText(columns[0].Template))

	assert.Equal(t, "name", columns[1].Name)
	assert.Equal(t, 17.0, columns[1].Width)
	assert.Equal(t, ALIGN_LEFT|WIDTH_EXACT, columns[1].Flags)
	assert.Equal(t, `{{index (.From | names) 0}}`,
		templateText(columns[1].Template))

	assert.Equal(t, "flags", columns[2].Name)
	assert.Equal(t, 4.0, columns[2].Width)
	assert.Equal(t, ALIGN_RIGHT|WIDTH_EXACT, columns[2].Flags)
	assert.Equal(t, `{{.Flags | join ""}}`,
		templateText(columns[2].Template))

	assert.Equal(t, "subject", columns[3].Name)
	assert.Equal(t, 0.0, columns[3].Width)
	assert.Equal(t, ALIGN_LEFT|WIDTH_AUTO, columns[3].Flags)
	assert.Equal(t, `{{.Subject}}`, templateText(columns[3].Template))
}
