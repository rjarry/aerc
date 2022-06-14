package parse_test

import (
	"io/ioutil"
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/parse"
)

func TestHyperlinks(t *testing.T) {
	tests := []struct {
		text  string
		links []string
	}{
		{
			text:  "http://aerc-mail.org",
			links: []string{"http://aerc-mail.org"},
		},
		{
			text:  "https://aerc-mail.org",
			links: []string{"https://aerc-mail.org"},
		},
		{
			text:  "text https://aerc-mail.org more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			text:  "text (https://aerc-mail.org) more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			text:  "text \"https://aerc-mail.org\" more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			text:  "text <https://aerc-mail.org> more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			text:  "<a href=\"https://aerc-mail.org\">",
			links: []string{"https://aerc-mail.org"},
		},
		{
			text:  "text https://aerc-mail.org more text https://aerc-mail.org more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			text:  "text https://aerc-mail.org more text http://git.sr.ht/~rjarry/aerc more text",
			links: []string{"https://aerc-mail.org", "http://git.sr.ht/~rjarry/aerc"},
		},
		{
			text:  "text http://www.ietf.org/rfc/rfc2396.txt more text",
			links: []string{"http://www.ietf.org/rfc/rfc2396.txt"},
		},
		{
			text:  "text <http://example.com:8042/over/there?name=ferret#nose> more text",
			links: []string{"http://example.com:8042/over/there?name=ferret#nose"},
		},
		{
			text:  "text http://cnn.example.com&story=breaking_news@10.0.0.1/top_story.htm more text",
			links: []string{"http://cnn.example.com&story=breaking_news@10.0.0.1/top_story.htm"},
		},
		{
			text:  "text https://www.ics.uci.edu/pub/ietf/uri/#Related more text",
			links: []string{"https://www.ics.uci.edu/pub/ietf/uri/#Related"},
		},
		{
			text:  "text https://www.example.com/index.php?id_sezione=360&sid=3a5ebc944f41daa6f849f730f1 more text",
			links: []string{"https://www.example.com/index.php?id_sezione=360&sid=3a5ebc944f41daa6f849f730f1"},
		},
	}

	for _, test := range tests {

		// make sure reader is exact copy of input reader
		reader, links := parse.HttpLinks(strings.NewReader(test.text))
		if data, err := ioutil.ReadAll(reader); err != nil {
			t.Errorf("could not read text: %v", err)
		} else if string(data) != test.text {
			t.Errorf("did not copy input reader correctly")
		}

		// check correct parsed links
		if len(links) != len(test.links) {
			t.Errorf("different number of links: got %d but expected %d", len(links), len(test.links))
		}
		linkMap := make(map[string]struct{})
		for _, got := range links {
			linkMap[got] = struct{}{}
		}
		for _, expected := range test.links {
			if _, ok := linkMap[expected]; !ok {
				t.Errorf("link not parsed: %s", expected)
			}
		}

	}
}
