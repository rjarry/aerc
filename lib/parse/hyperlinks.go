package parse

import (
	"bufio"
	"bytes"
	"io"
	"net/url"
	"regexp"
	"strings"
)

var urlRe = regexp.MustCompile(`([\w\d]{2,}:([^\s>\]\)"]|\][^\s>\)"]|\]$){8,})`)

// HttpLinks searches a reader for a http link and returns a copy of the
// reader and a slice with links.
func HttpLinks(r io.Reader) (io.Reader, []string) {
	var buf bytes.Buffer
	tr := io.TeeReader(r, &buf)

	scanner := bufio.NewScanner(tr)
	linkMap := make(map[string]struct{})
	for scanner.Scan() {
		line := scanner.Text()
		for _, word := range strings.Fields(line) {
			if links := urlRe.FindStringSubmatch(word); len(links) > 0 {
				if _, err := url.Parse(links[0]); err != nil {
					continue
				}
				linkMap[strings.TrimSpace(links[0])] = struct{}{}
			}
		}
	}

	results := []string{}
	for link := range linkMap {
		results = append(results, link)
	}

	return &buf, results
}
