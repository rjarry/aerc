package commands

import (
	"testing"
)

var parserTests = []struct {
	name       string
	cmd        string
	wantType   completionType
	wantFlag   string
	wantArg    string
	wantOptind int
}{
	{
		name:       "empty command",
		cmd:        "",
		wantType:   COMMAND,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 0,
	},
	{
		name:       "command only",
		cmd:        "cmd",
		wantType:   COMMAND,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 1,
	},
	{
		name:       "with space",
		cmd:        "cmd ",
		wantType:   OPERAND,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 1,
	},
	{
		name:       "with two spaces",
		cmd:        "cmd  ",
		wantType:   OPERAND,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 1,
	},
	{
		name:       "with single option flag",
		cmd:        "cmd -",
		wantType:   SHORT_OPTION,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with single option flag two spaces",
		cmd:        "cmd  -",
		wantType:   SHORT_OPTION,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with single option flag completed",
		cmd:        "cmd -a",
		wantType:   SHORT_OPTION,
		wantFlag:   "a",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with single option flag completed and space",
		cmd:        "cmd -a ",
		wantType:   OPERAND,
		wantFlag:   "a",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with single option flag completed and two spaces",
		cmd:        "cmd -a ",
		wantType:   OPERAND,
		wantFlag:   "a",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with two single option flag completed",
		cmd:        "cmd -b -a",
		wantType:   SHORT_OPTION,
		wantFlag:   "a",
		wantArg:    "",
		wantOptind: 3,
	},
	{
		name:       "with two single option flag combined",
		cmd:        "cmd -ab",
		wantType:   SHORT_OPTION,
		wantFlag:   "ab",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with two single option flag and space",
		cmd:        "cmd -ab ",
		wantType:   OPERAND,
		wantFlag:   "ab",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with mandatory option flag",
		cmd:        "cmd -f",
		wantType:   OPTION_ARGUMENT,
		wantFlag:   "f",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with mandatory option flag and space",
		cmd:        "cmd -f ",
		wantType:   OPTION_ARGUMENT,
		wantFlag:   "f",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with mandatory option flag and two spaces",
		cmd:        "cmd -f  ",
		wantType:   OPTION_ARGUMENT,
		wantFlag:   "f",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with mandatory option flag and completed",
		cmd:        "cmd -f a",
		wantType:   OPTION_ARGUMENT,
		wantFlag:   "f",
		wantArg:    "a",
		wantOptind: 3,
	},
	{
		name:       "with mandatory option flag and completed quote",
		cmd:        "cmd -f 'a b'",
		wantType:   OPTION_ARGUMENT,
		wantFlag:   "f",
		wantArg:    "a b",
		wantOptind: 3,
	},
	{
		name:       "with mandatory option flag and operand",
		cmd:        "cmd -f 'a b' hello",
		wantType:   OPERAND,
		wantFlag:   "f",
		wantArg:    "a b",
		wantOptind: 3,
	},
	{
		name:       "with mandatory option flag and two spaces between",
		cmd:        "cmd -f  a",
		wantType:   OPTION_ARGUMENT,
		wantFlag:   "f",
		wantArg:    "a",
		wantOptind: 3,
	},
	{
		name:       "with mandatory option flag and more spaces",
		cmd:        "cmd -f  a ",
		wantType:   OPERAND,
		wantFlag:   "f",
		wantArg:    "a",
		wantOptind: 3,
	},
	{
		name:       "with template data",
		cmd:        "cmd -a {{if .Size}}  hello {{else}} {{end}}",
		wantType:   OPERAND,
		wantFlag:   "a",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with operand",
		cmd:        "cmd -ab /tmp/aerc-",
		wantType:   OPERAND,
		wantFlag:   "ab",
		wantArg:    "",
		wantOptind: 2,
	},
	{
		name:       "with operand indicator",
		cmd:        "cmd -ab -- /tmp/aerc-",
		wantType:   OPERAND,
		wantFlag:   "ab",
		wantArg:    "",
		wantOptind: 3,
	},
	{
		name:       "hyphen connected command",
		cmd:        "cmd-dmc",
		wantType:   COMMAND,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 1,
	},
	{
		name:       "incomplete hyphen connected command",
		cmd:        "cmd-",
		wantType:   COMMAND,
		wantFlag:   "",
		wantArg:    "",
		wantOptind: 1,
	},
	{
		name:       "hyphen connected command with option",
		cmd:        "cmd-dmc -a",
		wantType:   SHORT_OPTION,
		wantFlag:   "a",
		wantArg:    "",
		wantOptind: 2,
	},
}

func TestCommands_Parser(t *testing.T) {
	for i, test := range parserTests {
		n := len(test.cmd)
		spaceTerminated := n > 0 && test.cmd[n-1] == ' '
		parser, err := newParser(test.cmd, "abf:", spaceTerminated)
		if err != nil {
			t.Errorf("parser error: %v", err)
		}

		if test.wantType != parser.kind {
			t.Errorf("test %d '%s': completion type does not match: "+
				"want %d, but got %d", i, test.cmd, test.wantType,
				parser.kind)
		}

		if test.wantFlag != parser.flag {
			t.Errorf("test %d '%s': flag does not match: "+
				"want %s, but got %s", i, test.cmd, test.wantFlag,
				parser.flag)
		}

		if test.wantArg != parser.arg {
			t.Errorf("test %d '%s': arg does not match: "+
				"want %s, but got %s", i, test.cmd, test.wantArg,
				parser.arg)
		}

		if test.wantOptind != parser.optind {
			t.Errorf("test %d '%s': optind does not match: "+
				"want %d, but got %d", i, test.cmd, test.wantOptind,
				parser.optind)
		}
	}
}
