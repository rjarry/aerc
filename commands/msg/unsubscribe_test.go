package msg

import (
	"testing"
)

func TestParseUnsubscribe(t *testing.T) {
	type tc struct {
		hdr      string
		expected []string
	}
	cases := []*tc{
		&tc{"", []string{}},
		&tc{"invalid", []string{}},
		&tc{"<https://example.com>, <http://example.com>", []string{
			"https://example.com", "http://example.com",
		}},
		&tc{"<https://example.com> is a URL", []string{
			"https://example.com",
		}},
		&tc{"<mailto:user@host?subject=unsubscribe>, <https://example.com>",
			[]string{
				"mailto:user@host?subject=unsubscribe", "https://example.com",
			}},
		&tc{"<>, <https://example> ", []string{
			"", "https://example",
		}},
	}
	for _, c := range cases {
		result := parseUnsubscribeMethods(c.hdr)
		if len(result) != len(c.expected) {
			t.Errorf("expected %d methods but got %d", len(c.expected), len(result))
			continue
		}
		for idx := 0; idx < len(result); idx++ {
			if result[idx].String() != c.expected[idx] {
				t.Errorf("expected %v but got %v", c.expected[idx], result[idx])
			}
		}
	}
}
