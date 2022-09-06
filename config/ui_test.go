package config

import (
	"bytes"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"github.com/stretchr/testify/assert"
)

func TestConvertIndexFormat(t *testing.T) {
	columns, err := convertIndexFormat("%-20.20D %-17.17n %Z %s")
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, columns, 4)

	data := templates.DummyData()
	var buf bytes.Buffer

	assert.Equal(t, "date", columns[0].Name)
	assert.Equal(t, 20.0, columns[0].Width)
	assert.Equal(t, ALIGN_LEFT|WIDTH_EXACT, columns[0].Flags)
	assert.Nil(t, columns[0].Template.Execute(&buf, data))

	buf.Reset()
	assert.Equal(t, "name", columns[1].Name)
	assert.Equal(t, 17.0, columns[1].Width)
	assert.Equal(t, ALIGN_LEFT|WIDTH_EXACT, columns[1].Flags)
	assert.Nil(t, columns[1].Template.Execute(&buf, data))
	assert.Equal(t, "John Doe", buf.String())

	buf.Reset()
	assert.Equal(t, "flags", columns[2].Name)
	assert.Equal(t, 4.0, columns[2].Width)
	assert.Equal(t, ALIGN_RIGHT|WIDTH_EXACT, columns[2].Flags)
	assert.Nil(t, columns[2].Template.Execute(&buf, data))
	assert.Equal(t, "O!*", buf.String())

	buf.Reset()
	assert.Equal(t, "subject", columns[3].Name)
	assert.Equal(t, 0.0, columns[3].Width)
	assert.Equal(t, ALIGN_LEFT|WIDTH_AUTO, columns[3].Flags)
	assert.Nil(t, columns[3].Template.Execute(&buf, data))
	assert.Equal(t, "[PATCH aerc 2/3] foo: baz bar buz", buf.String())
}
