package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMessageNoFilename(t *testing.T) {
	assert.Equal(t, "mime/type", formatMessagePart("mime/type", "", 24))
	assert.Equal(t, "m/type", formatMessagePart("m/type", "", 24))
	assert.Equal(t, "2", formatMessagePart("2", "", 24))
	assert.Equal(t, "2", formatMessagePart("2", "", 20))
}

func TestFormatMessageNoFilenameNotEnoguhSpace(t *testing.T) {
	assert.Equal(t, "mime/type", formatMessagePart("mime/type", "", 24))
	assert.Equal(t, "mime/type", formatMessagePart("mime/type", "", 9))
	assert.Equal(t, "mime/ty…", formatMessagePart("mime/type", "", 8))
	assert.Equal(t, "mime/…", formatMessagePart("mime/type", "", 6))
	assert.Equal(t, "m…", formatMessagePart("mime/type", "", 2))
	assert.Equal(t, "…", formatMessagePart("mime/type", "", 1))

	assert.Equal(t, "", formatMessagePart("mime/type", "", 0))
	assert.Equal(t, "", formatMessagePart("mime/type", "", -1))
	assert.Equal(t, "", formatMessagePart("mime/type", "", -10))
}

func TestFormatMessagePartSimpleCases(t *testing.T) {
	assert.Equal(t, "filename.doc (mime/type)", formatMessagePart("mime/type", "filename.doc", 24))
	assert.Equal(t, "имяфайла.док (mime/type)", formatMessagePart("mime/type", "имяфайла.док", 24))
	assert.Equal(t, "file.doc        (m/type)", formatMessagePart("m/type", "file.doc", 24))
	assert.Equal(t, "1                    (2)", formatMessagePart("2", "1", 24))
	assert.Equal(t, "1                (2)", formatMessagePart("2", "1", 20))
	assert.Equal(t, "1 (2)", formatMessagePart("2", "1", 5))
}

func TestFormatMessagePartNotEnoughSpaceForMime(t *testing.T) {
	assert.Equal(t, "filename.doc       (mime/type)", formatMessagePart("mime/type", "filename.doc", 30))
	assert.Equal(t, "filename.doc  (mime/type)", formatMessagePart("mime/type", "filename.doc", 25))
	assert.Equal(t, "filename.doc (mime/type)", formatMessagePart("mime/type", "filename.doc", 24))
	assert.Equal(t, "filename.doc (mime/ty…)", formatMessagePart("mime/type", "filename.doc", 23))
	assert.Equal(t, "имяфайла.док (mime/ty…)", formatMessagePart("mime/type", "имяфайла.док", 23))
	assert.Equal(t, "filename.doc (m…)", formatMessagePart("mime/type", "filename.doc", 17))
	assert.Equal(t, "filename.doc (…)", formatMessagePart("mime/type", "filename.doc", 16))
	assert.Equal(t, "имяфайла.док (…)", formatMessagePart("mime/type", "имяфайла.док", 16))
	assert.Equal(t, "filename.doc", formatMessagePart("mime/type", "filename.doc", 15))
	assert.Equal(t, "filename.doc", formatMessagePart("mime/type", "filename.doc", 14))
	assert.Equal(t, "filename.doc", formatMessagePart("mime/type", "filename.doc", 13))
	assert.Equal(t, "filename.doc", formatMessagePart("mime/type", "filename.doc", 12))
	assert.Equal(t, "имяфайла.док", formatMessagePart("mime/type", "имяфайла.док", 12))
}

func TestFormatMessagePartNotEnoughSpaceForFilename(t *testing.T) {
	assert.Equal(t, "filename.d…", formatMessagePart("mime/type", "filename.doc", 11))
	assert.Equal(t, "filename…", formatMessagePart("mime/type", "filename.doc", 9))
	assert.Equal(t, "f…", formatMessagePart("mime/type", "filename.doc", 2))
	assert.Equal(t, "…", formatMessagePart("mime/type", "filename.doc", 1))

	assert.Equal(t, "", formatMessagePart("mime/type", "filename.doc", 0))
	assert.Equal(t, "", formatMessagePart("mime/type", "filename.doc", -1))
	assert.Equal(t, "", formatMessagePart("mime/type", "filename.doc", -10))

	assert.Equal(t, "имяфайла.д…", formatMessagePart("mime/type", "имяфайла.док", 11))
	assert.Equal(t, "имяфайла…", formatMessagePart("mime/type", "имяфайла.док", 9))
	assert.Equal(t, "и…", formatMessagePart("mime/type", "имяфайла.док", 2))
	assert.Equal(t, "…", formatMessagePart("mime/type", "имяфайла.док", 1))
}
