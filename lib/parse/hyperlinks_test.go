package parse_test

import (
	"io"
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/parse"
)

func TestHyperlinks(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		links []string
	}{
		{
			name:  "http-link",
			text:  "http://aerc-mail.org",
			links: []string{"http://aerc-mail.org"},
		},
		{
			name:  "https-link",
			text:  "https://aerc-mail.org",
			links: []string{"https://aerc-mail.org"},
		},
		{
			name:  "https-link-in-text",
			text:  "text https://aerc-mail.org more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			name:  "https-link-in-parenthesis",
			text:  "text (https://aerc-mail.org) more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			name:  "https-link-in-quotes",
			text:  "text \"https://aerc-mail.org\" more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			name:  "https-link-in-angle-brackets",
			text:  "text <https://aerc-mail.org> more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			name:  "https-link-in-html",
			text:  "<a href=\"https://aerc-mail.org\">",
			links: []string{"https://aerc-mail.org"},
		},
		{
			name:  "https-link-twice",
			text:  "text https://aerc-mail.org more text https://aerc-mail.org more text",
			links: []string{"https://aerc-mail.org"},
		},
		{
			name:  "multiple-links",
			text:  "text https://aerc-mail.org more text http://git.sr.ht/~rjarry/aerc more text",
			links: []string{"https://aerc-mail.org", "http://git.sr.ht/~rjarry/aerc"},
		},
		{
			name:  "rfc",
			text:  "text http://www.ietf.org/rfc/rfc2396.txt more text",
			links: []string{"http://www.ietf.org/rfc/rfc2396.txt"},
		},
		{
			name:  "http-with-query-and-fragment",
			text:  "text <http://example.com:8042/over/there?name=ferret#nose> more text",
			links: []string{"http://example.com:8042/over/there?name=ferret#nose"},
		},
		{
			name:  "http-with-at",
			text:  "text http://cnn.example.com&story=breaking_news@10.0.0.1/top_story.htm more text",
			links: []string{"http://cnn.example.com&story=breaking_news@10.0.0.1/top_story.htm"},
		},
		{
			name:  "https-with-fragment",
			text:  "text https://www.ics.uci.edu/pub/ietf/uri/#Related more text",
			links: []string{"https://www.ics.uci.edu/pub/ietf/uri/#Related"},
		},
		{
			name:  "https-with-query",
			text:  "text https://www.example.com/index.php?id_sezione=360&sid=3a5ebc944f41daa6f849f730f1 more text",
			links: []string{"https://www.example.com/index.php?id_sezione=360&sid=3a5ebc944f41daa6f849f730f1"},
		},
		{
			name:  "https-onedrive",
			text:  "I have a link like this in an email (I deleted a few characters here-and-there for privacy) https://1drv.ms/w/s!Ap-KLfhNxS4fRt6tIvw?e=dW8WLO",
			links: []string{"https://1drv.ms/w/s!Ap-KLfhNxS4fRt6tIvw?e=dW8WLO"},
		},
		{
			name:  "mailto-ipv6",
			text:  "You can reach me via the somewhat strange, but nonetheless valid, email mailto:~mpldr/list@[2001:db8::7]",
			links: []string{"mailto:~mpldr/list@[2001:db8::7]"},
		},
		{
			name:  "mailto-ipv6-query",
			text:  "You can reach me via the somewhat strange, but nonetheless valid, email mailto:~mpldr/list@[2001:db8::7]?subject=whazzup%3F",
			links: []string{"mailto:~mpldr/list@[2001:db8::7]?subject=whazzup%3F"},
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// make sure reader is exact copy of input reader
			reader, parsedLinks := parse.HttpLinks(strings.NewReader(test.text))
			if _, err := io.ReadAll(reader); err != nil {
				t.Skipf("could not read text: %v", err)
			}

			// check correct parsed links
			if len(parsedLinks) != len(test.links) {
				t.Errorf("different number of links: got %d but expected %d", len(parsedLinks), len(test.links))
			}
			linkMap := make(map[string]struct{})
			for _, got := range parsedLinks {
				linkMap[got] = struct{}{}
			}
			for _, expected := range test.links {
				if _, ok := linkMap[expected]; !ok {
					t.Errorf("link[%d] not parsed: %s", i, expected)
				}
			}
		})
	}
}
