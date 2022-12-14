package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

type paragraph struct {
	// email quote prefix, if any
	quotes string
	// list item indent, if any
	leader string
	// actual text of this paragraph
	text string
	// percentage of letters in text
	proseRatio int
	// text ends with a space
	flowed bool
	// paragraph is a list item
	listItem bool
}

func main() {
	var err error
	var width int
	var reflow bool
	var file string
	var proseRatio int
	var input *os.File

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&width, "w", 80, "preferred wrap margin")
	fs.BoolVar(&reflow, "r", false,
		"reflow all paragraphs even if no trailing space")
	fs.IntVar(&proseRatio, "l", 50,
		"minimum percentage of letters in a line to be considered a paragaph")
	fs.StringVar(&file, "f", "", "read from file instead of stdin")
	_ = fs.Parse(os.Args[1:])

	if file != "" {
		input, err = os.OpenFile(file, os.O_RDONLY, 0o644)
		if err != nil {
			goto end
		}
	} else {
		input = os.Stdin
	}

	err = wrap(input, os.Stdout, width, reflow, proseRatio)

end:
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func wrap(
	in io.Reader, out io.Writer, width int, reflow bool, proseRatio int,
) error {
	var para *paragraph = nil
	var line string
	var err error

	if patchSubjectRe.MatchString(os.Getenv("AERC_SUBJECT")) {
		// never reflow patches
		_, err = io.Copy(out, in)
	} else {
		reader := bufio.NewReader(in)
		line, err = reader.ReadString('\n')
		for ; err == nil; line, err = reader.ReadString('\n') {
			next := parse(line)
			switch {
			case para == nil:
				para = next
			case para.isContinuation(next, reflow, proseRatio):
				para.join(next)
			default:
				para.write(out, width, proseRatio)
				para = next
			}
		}
		if para != nil {
			para.write(out, width, proseRatio)
		}
	}

	return err
}

// Parse a line of text into a paragraph structure
func parse(line string) *paragraph {
	p := new(paragraph)
	q := 0
	t := 0
	line = strings.TrimRight(line, "\r\n")
	// tabs cause a whole lot of troubles, replace them with 8 spaces
	line = strings.ReplaceAll(line, "\t", "        ")

	// Use integer offsets to find relevant positions in the line
	//
	// > > >      2)        blah blah blah blah
	//       ^--------+-----^
	//       q        |     t
	//  end of quotes |  start of text
	//                |
	//          list item leader

	// detect the end of quotes prefix if any
	for q < len(line) && line[q] == '>' {
		q += 1
		if q < len(line) && line[q] == ' ' {
			q += 1
		}
	}

	// detect list item leader
	loc := listItemRe.FindStringIndex(line[q:])
	if loc != nil {
		// start of list item
		p.listItem = true
	} else {
		// maybe list item continuation
		loc = leadingSpaceRe.FindStringIndex(line[q:])
	}
	if loc != nil {
		t = q + loc[1]
	} else {
		// no list at all
		t = q
	}

	// check if there is trailing whitespace, indicating format=flowed
	loc = trailingSpaceRe.FindStringIndex(line[t:])
	if loc != nil {
		p.flowed = true
		// trim whitespace
		line = line[:t+loc[0]]
	}

	p.quotes = line[:q]
	p.leader = strings.Repeat(" ", runewidth.StringWidth(line[q:t]))
	p.text = line[q:]

	// compute the ratio of letters in the actual text
	onlyLetters := strings.TrimLeft(line[q:], " ")
	totalLen := runewidth.StringWidth(onlyLetters)
	if totalLen == 0 {
		// to avoid division by zero
		totalLen = 1
	}
	onlyLetters = notLetterRe.ReplaceAllLiteralString(onlyLetters, "")
	p.proseRatio = 100 * runewidth.StringWidth(onlyLetters) / totalLen

	return p
}

// Return true if a paragraph is a continuation of the current one.
func (p *paragraph) isContinuation(
	next *paragraph, reflow bool, proseRatio int,
) bool {
	switch {
	case next.listItem:
		// new list items always start a new paragraph
		return false
	case next.proseRatio < proseRatio || p.proseRatio < proseRatio:
		// does not look like prose, maybe ascii art
		return false
	case next.quotes != p.quotes || next.leader != p.leader:
		// quote level and/or list item leader have changed
		return false
	case len(strings.Trim(next.text, " ")) == 0:
		// empty line
		return false
	case p.flowed:
		// current paragraph has trailing space, indicating
		// format=flowed
		return true
	case reflow:
		// user forced paragraph reflow on the command line
		return true
	default:
		return false
	}
}

// Join next paragraph into current one.
func (p *paragraph) join(next *paragraph) {
	if p.text == "" {
		p.text = next.text
	} else {
		p.text = p.text + " " + strings.Trim(next.text, " ")
	}
	p.proseRatio = (p.proseRatio + next.proseRatio) / 2
	p.flowed = next.flowed
}

// Write a paragraph, wrapping at words boundaries.
//
// Only try to do word wrapping on things that look like prose. When the text
// contains too many non-letter characters, print it as-is.
func (p *paragraph) write(out io.Writer, margin int, proseRatio int) {
	leader := ""
	more := true
	quotesWidth := runewidth.StringWidth(p.quotes)
	for more {
		var line string
		width := quotesWidth + runewidth.StringWidth(leader)
		remain := runewidth.StringWidth(p.text)
		if width+remain <= margin || p.proseRatio < proseRatio {
			// whole paragraph fits on a single line
			line = p.text
			p.text = ""
			more = false
		} else {
			// find split point, preferably before margin
			split := -1
			w := 0
			for i, r := range p.text {
				w += runewidth.RuneWidth(r)
				if width+w > margin && split != -1 {
					break
				}
				if r == ' ' {
					split = i
				}
			}
			if split == -1 {
				// no space found to split, print a long line
				line = p.text
				p.text = ""
				more = false
			} else {
				line = p.text[:split]
				// find start of next word
				for split < len(p.text) && p.text[split] == ' ' {
					split++
				}
				if split < len(p.text) {
					p.text = p.text[split:]
				} else {
					// only trailing whitespace, we're done
					p.text = ""
					more = false
				}
			}
		}
		fmt.Fprintf(out, "%s%s%s\n", p.quotes, leader, line)
		leader = p.leader
	}
}

var (
	patchSubjectRe  = regexp.MustCompile(`\bPATCH\b`)
	listItemRe      = regexp.MustCompile(`^\s*([\-\*\.]|[a-z\d]{1,2}[\)\]\.])\s+`)
	leadingSpaceRe  = regexp.MustCompile(`^\s+`)
	trailingSpaceRe = regexp.MustCompile(`\s+$`)
	notLetterRe     = regexp.MustCompile(`[^\pL]`)
)
