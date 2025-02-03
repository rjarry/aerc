package parse

import (
	"bytes"
	"io"
	"regexp"
	"sort"
)

// Partial regexp to match the beginning of URLs and email addresses.
// The remainder of the matched URLs/emails is parsed manually.
var urlRe = regexp.MustCompile(
	`([a-z]{2,8})://` + // URL start
		`|` + // or
		`(mailto:)?[[:alnum:]_+.~/-]*[[:alnum:]]@`, // email start
)

// HttpLinks searches a reader for a http link and returns a copy of the
// reader and a slice with links. If isHtml is true, left angle brackets are
// considered to always be right link delimiters.
func HttpLinks(r io.Reader, isHtml bool) (io.Reader, []string) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return r, nil
	}

	links := make(map[string]struct{})
	b := buf
	match := urlRe.FindSubmatchIndex(b)
	for ; match != nil; match = urlRe.FindSubmatchIndex(b) {
		// Regular expressions do not really cut it here and we
		// need to detect opening/closing braces to handle
		// markdown link syntax.
		var paren, bracket, ltgt, scheme int
		var emitUrl bool
		i, j := match[0], match[1]
		b = b[i:]
		scheme = j - i
		j = scheme

		// "inline" email without a mailto: prefix - add some extra checks for those
		inlineEmail := len(match) > 4 && match[2] == -1 && match[4] == -1

		for !emitUrl && j < len(b) && bytes.IndexByte(urichars, b[j]) != -1 {
			switch b[j] {
			case '[':
				bracket++
				j++
			case '(':
				paren++
				j++
			case '<':
				if isHtml {
					emitUrl = true
				} else {
					ltgt++
					j++
				}
			case ']':
				bracket--
				if bracket < 0 {
					emitUrl = true
				} else {
					j++
				}
			case ')':
				paren--
				if paren < 0 {
					emitUrl = true
				} else {
					j++
				}
			case '>':
				ltgt--
				if ltgt < 0 {
					emitUrl = true
				} else {
					j++
				}
			case '&':
				if inlineEmail {
					emitUrl = true
				} else {
					j++
				}
			default:
				j++
			}

			// we don't want those in inline emails
			if inlineEmail && (paren > 0 || ltgt > 0 || bracket > 0) {
				j--
				emitUrl = true
			}
		}

		// Heuristic to remove trailing characters that are
		// valid URL characters, but typically not at the end of
		// the URL
		for trim := true; trim && j > 0; {
			switch b[j-1] {
			case '.', ',', ':', ';', '?', '!', '"', '\'', '%':
				j--
			default:
				trim = false
			}
		}
		if j == scheme {
			// Only an URL scheme, ignore.
			b = b[j:]
			continue
		}
		url := string(b[:j])
		if inlineEmail {
			// Email address with missing mailto: scheme. Add it.
			url = "mailto:" + url
		}
		links[url] = struct{}{}
		b = b[j:]
	}

	results := make([]string, 0, len(links))
	for link := range links {
		results = append(results, link)
	}
	sort.Strings(results)

	return bytes.NewReader(buf), results
}

var urichars = []byte(
	"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789-_.,~:;/?#@!$&%*+=\"'<>()[]",
)
