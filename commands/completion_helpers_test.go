package commands_test

import (
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/commands"
)

func TestCommands_Operand(t *testing.T) {
	tests := []struct {
		args []string
		spec string
		want string
	}{
		{
			args: []string{"cmd", "-a", "-b", "arg1", "-c", "bla"},
			spec: "ab:c",
			want: "cmdbla",
		},
		{
			args: []string{"cmd", "-a", "-b", "arg1", "-c", "--", "bla"},
			spec: "ab:c",
			want: "bla",
		},
		{
			args: []string{"cmd", "-a", "-b", "arg1", "-c", "bla"},
			spec: "ab:c:",
			want: "cmd",
		},
		{
			args: nil,
			spec: "ab:c:",
			want: "",
		},
	}
	for i, test := range tests {
		arg := strings.Join(commands.Operands(test.args, test.spec), "")
		if arg != test.want {
			t.Errorf("failed test %d: want '%s', got '%s'", i,
				test.want, arg)
		}
	}
}
