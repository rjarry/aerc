package middleware

import (
	"reflect"
	"testing"
)

func TestFolderMap_Apply(t *testing.T) {
	tests := []struct {
		name    string
		mapping map[string]string
		order   []string
		input   []string
		want    []string
	}{
		{
			name:    "strict single folder mapping",
			mapping: map[string]string{"Drafts": "INBOX/Drafts"},
			order:   []string{"Drafts"},
			input:   []string{"INBOX/Drafts"},
			want:    []string{"Drafts"},
		},
		{
			name:    "prefix mapping with * suffix",
			mapping: map[string]string{"Prefix/": "INBOX/*"},
			order:   []string{"Prefix/"},
			input:   []string{"INBOX", "INBOX/Test1", "INBOX/Test2", "Archive"},
			want:    []string{"INBOX", "Prefix/Test1", "Prefix/Test2", "Archive"},
		},
		{
			name:    "remove prefix with * in key",
			mapping: map[string]string{"*": "INBOX/*"},
			order:   []string{"*"},
			input:   []string{"INBOX", "INBOX/Test1", "INBOX/Test2", "Archive"},
			want:    []string{"INBOX", "Test1", "Test2", "Archive"},
		},
		{
			name: "remove two prefixes with * in keys",
			mapping: map[string]string{
				"*":  "INBOX/*",
				"**": "PROJECT/*",
			},
			order: []string{"*", "**"},
			input: []string{"INBOX", "INBOX/Test1", "INBOX/Test2", "Archive", "PROJECT/sub1", "PROJECT/sub2"},
			want:  []string{"INBOX", "Test1", "Test2", "Archive", "sub1", "sub2"},
		},
		{
			name: "multiple, sequential mappings",
			mapping: map[string]string{
				"Archive/existing": "Archive*",
				"Archive":          "Archivum*",
			},
			order: []string{"Archive/existing", "Archive"},
			input: []string{"Archive", "Archive/sub", "Archivum", "Archivum/year1"},
			want:  []string{"Archive/existing", "Archive/existing/sub", "Archive", "Archive/year1"},
		},
	}

	for i, test := range tests {
		fm := &folderMap{
			mapping: test.mapping,
			order:   test.order,
		}
		var result []string
		for _, in := range test.input {
			result = append(result, fm.Apply(in))
		}
		if !reflect.DeepEqual(result, test.want) {
			t.Errorf("test (%d: %s) failed: want '%v' but got '%v'",
				i, test.name, test.want, result)
		}
	}
}

func TestFolderMap_createFolder(t *testing.T) {
	tests := []struct {
		name  string
		table map[string]string
		input string
		want  string
	}{
		{
			name:  "create normal folder",
			table: map[string]string{"Drafts": "INBOX/Drafts"},
			input: "INBOX/Drafts2",
			want:  "INBOX/Drafts2",
		},
		{
			name:  "create mapped folder",
			table: map[string]string{"Drafts": "INBOX/Drafts"},
			input: "Drafts/Sub",
			want:  "INBOX/Drafts/Sub",
		},
	}

	for i, test := range tests {
		result := createFolder(test.table, test.input)
		if result != test.want {
			t.Errorf("test (%d: %s) failed: want '%v' but got '%v'",
				i, test.name, test.want, result)
		}
	}
}
