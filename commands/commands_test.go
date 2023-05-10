package commands

import (
	"reflect"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/state"
)

func TestExecuteCommand_expand(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{
			args: []string{"prompt", "Really quit? ", "quit"},
			want: []string{"prompt", "Really quit? ", "quit"},
		},
		{
			args: []string{"{{", "print", "\"hello\"", "}}"},
			want: []string{"hello"},
		},
		{
			args: []string{"prompt", "Really quit  ? ", "  quit "},
			want: []string{"prompt", "Really quit  ? ", "  quit "},
		},
		{
			args: []string{
				"prompt", "Really quit? ", "{{",
				"print", "\"quit\"", "}}",
			},
			want: []string{"prompt", "Really quit? ", "quit"},
		},
		{
			args: []string{
				"prompt", "Really quit? ", "{{",
				"if", "1", "}}", "quit", "{{end}}",
			},
			want: []string{"prompt", "Really quit? ", "quit"},
		},
	}

	data := state.TemplateData{}

	for i, test := range tests {
		got, err := expand(&data, test.args)
		if err != nil {
			t.Errorf("test %d failed with err: %v", i, err)
		} else if !reflect.DeepEqual(got, test.want) {
			t.Errorf("test %d failed: "+
				"got: %v, but want: %v", i, got, test.want)
		}
	}
}
