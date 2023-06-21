package lib_test

import (
	"reflect"
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/worker/lib"
)

func TestFolderMap(t *testing.T) {
	text := `#this is comment

	   Sent =    [Gmail]/Sent

	    # a comment between entries
	Spam=[Gmail]/Spam # this is comment after the values
	`
	fmap, order, err := lib.ParseFolderMap(strings.NewReader(text))
	if err != nil {
		t.Errorf("parsing failed: %v", err)
	}

	want_map := map[string]string{
		"Sent": "[Gmail]/Sent",
		"Spam": "[Gmail]/Spam",
	}
	want_order := []string{"Sent", "Spam"}

	if !reflect.DeepEqual(order, want_order) {
		t.Errorf("order is not correct; want: %v, got: %v",
			want_order, order)
	}

	if !reflect.DeepEqual(fmap, want_map) {
		t.Errorf("map is not correct; want: %v, got: %v",
			want_map, fmap)
	}
}

func TestFolderMap_ExpectFails(t *testing.T) {
	tests := []string{
		`key = `,
		` = value`,
		` = `,
		`key = #value`,
	}
	for _, text := range tests {
		_, _, err := lib.ParseFolderMap(strings.NewReader(text))
		if err == nil {
			t.Errorf("expected to fail, but it did not: %v", text)
		}
	}
}
