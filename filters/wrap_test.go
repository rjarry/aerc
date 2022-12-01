package main

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type vector struct {
	name   string
	in     string
	out    string
	width  int
	reflow bool
	ratio  int
}

var vectors = []vector{
	{
		name: "simple",
		in: `long line that exceeds margin by many words
`,
		width:  30,
		reflow: false,
		ratio:  50,
		out: `long line that exceeds margin
by many words
`,
	},
	{
		name: "two-paragraphs",
		in: `this is one long paragraph
this is another long one
`,
		width:  20,
		reflow: false,
		ratio:  50,
		out: `this is one long
paragraph
this is another
long one
`,
	},
	{
		name: "reflow",
		in: `this is one long paragraph
this is another long one
`,
		width:  20,
		reflow: true,
		ratio:  50,
		out: `this is one long
paragraph this is
another long one
`,
	},
	{
		name: "quotes",
		in: `Let's play with quotes:

>> Hi there how are you doing?
> Great thanks

How rude.

>> Fantastic. Let's go wrap some words.
`,
		width:  20,
		reflow: false,
		ratio:  50,
		out: `Let's play with
quotes:

>> Hi there how are
>> you doing?
> Great thanks

How rude.

>> Fantastic. Let's
>> go wrap some
>> words.
`,
	},
	{
		name: "ascii-art",
		in: `This is a nice drawing, isn't it?

+-------------------+
|      foobaz       |
+-------------------+
          |
          |
+-------------------+
|      foobar       |
+-------------------+
`,
		width:  15,
		ratio:  50,
		reflow: true,
		out: `This is a nice
drawing, isn't
it?

+-------------------+
|      foobaz       |
+-------------------+
          |
          |
+-------------------+
|      foobar       |
+-------------------+
`,
	},
	{
		name: "list-items",
		in: `Shopping list:

  -  milk
  -  chocolate
  -  cookies (please, with nuts)
`,
		width:  20,
		reflow: false,
		ratio:  50,
		out: `Shopping list:

  -  milk
  -  chocolate
  -  cookies
     (please, with
     nuts)
`,
	},
	{
		name: "list-items-reflow",
		in: `Shopping list:

  *  milk
  *  chocolate
  *  cookies
     (please,
     with nuts)
`,
		width:  100,
		reflow: true,
		ratio:  30,
		out: `Shopping list:

  *  milk
  *  chocolate
  *  cookies (please, with nuts)
`,
	},
	{
		name: "long-url",
		in: `Please follow this ugly link:
http://foobaz.org/xapapzolmkdmldfk-fldskjflsk-cisjoij/onoes.jsp?xxx=2&yyy=3
`,
		width:  20,
		reflow: true,
		ratio:  50,
		out: `Please follow this
ugly link:
http://foobaz.org/xapapzolmkdmldfk-fldskjflsk-cisjoij/onoes.jsp?xxx=2&yyy=3
`,
	},
	{
		name:   "format=flowed",
		in:     "Oh, \nI'm \nso \nhip \nI \nuse \nformat=flowed.\n",
		width:  30,
		reflow: false,
		ratio:  50,
		out:    "Oh, I'm so hip I use\nformat=flowed.\n",
	},
	{
		name: "non-ascii",
		in: `Lorem ççççç ççççç ççç ççççç çç ççç ççççç çççççççç ççç çç ççççç ççççççççççç ççççç

Lorem жжжжж жжжжж жжж жжжжж жж жжж жжжжж жжжжжжжж жжж жж жжжжж жжжжжжжжжжж жжжжж жжжжжжжж
`,
		width:  40,
		reflow: false,
		ratio:  50,
		out: `Lorem ççççç ççççç ççç ççççç çç ççç
ççççç çççççççç ççç çç ççççç ççççççççççç
ççççç

Lorem жжжжж жжжжж жжж жжжжж жж жжж
жжжжж жжжжжжжж жжж жж жжжжж жжжжжжжжжжж
жжжжж жжжжжжжж
`,
	},
}

func TestWrap(t *testing.T) {
	for _, vec := range vectors {
		t.Run(vec.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(vec.in))
			var buf bytes.Buffer
			err := wrap(r, &buf, vec.width, vec.reflow, vec.ratio)
			if err != nil && !errors.Is(err, io.EOF) {
				t.Fatalf("[%s]: %v", vec.name, err)
			}
			if buf.String() != vec.out {
				t.Errorf("[%s] invalid format:\n%q\nexpected\n%q",
					vec.name, buf.String(), vec.out)
			}
		})
	}
}
