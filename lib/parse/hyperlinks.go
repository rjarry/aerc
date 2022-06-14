package parse

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"
)

var submatch = `(https?:\/\/[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,10}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&\/=]*))`
var httpRe = regexp.MustCompile("\"" + submatch + "\"" + "|" + "\\(" + submatch + "\\)" + "|" + "<" + submatch + ">" + "|" + submatch)

// HttpLinks searches a reader for a http link and returns a copy of the
// reader and a slice with links.
func HttpLinks(r io.Reader) (io.Reader, []string) {
	var buf bytes.Buffer
	tr := io.TeeReader(r, &buf)

	scanner := bufio.NewScanner(tr)
	linkMap := make(map[string]struct{})
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "http") {
			continue
		}
		for _, word := range strings.Fields(line) {
			if links := httpRe.FindStringSubmatch(word); len(links) > 0 {
				for _, l := range links[1:] {
					if l != "" {
						linkMap[strings.TrimSpace(l)] = struct{}{}
					}
				}
			}
		}
	}

	results := []string{}
	for link, _ := range linkMap {
		results = append(results, link)
	}

	return &buf, results
}
