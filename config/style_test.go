package config

import (
	"testing"

	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

const multiHeaderStyleset string = `
msglist_*.fg = salmon
msglist_*.From,~^"Bob Foo".fg = khaki
msglist_*.From,~^"Bob Foo".selected.fg = palegreen
msglist_*.Subject,~PATCH.From,~^"Bob Foo".fg = coral
msglist_*.From,~^"Bob Foo".Subject,~PATCH.X-Baz,exact.X-Clacks-Overhead,~Pratchett$.fg = plum
msglist_*.From,~^"Bob Foo".Subject,~PATCH.X-Clacks-Overhead,~Pratchett$.fg = pink
`

func TestStyleMultiHeaderPattern(t *testing.T) {
	ini, err := ini.Load([]byte(multiHeaderStyleset))
	if err != nil {
		t.Errorf("failed to load styleset: %v", err)
	}

	ss := NewStyleSet()
	err = ss.ParseStyleSet(ini)
	if err != nil {
		t.Errorf("failed to parse styleset: %v", err)
	}

	t.Run("default color", func(t *testing.T) {
		var h mail.Header
		h.SetAddressList("From", []*mail.Address{{"Alice Foo", "alice@foo.org"}})

		s := ss.Get(STYLE_MSGLIST_DEFAULT, &h)
		if s.Foreground != colorNames["salmon"] {
			t.Errorf("expected:#%v got:#%v", colorNames["salmon"], s.Foreground)
		}
	})

	t.Run("single header", func(t *testing.T) {
		var h mail.Header
		h.SetAddressList("From", []*mail.Address{{"Bob Foo", "bob@foo.org"}})

		s := ss.Get(STYLE_MSGLIST_DEFAULT, &h)
		if s.Foreground != colorNames["khaki"] {
			t.Errorf("expected:#%v got:#%v", colorNames["khaki"], s.Foreground)
		}
	})

	t.Run("two headers", func(t *testing.T) {
		var h mail.Header
		h.SetAddressList("From", []*mail.Address{{"Bob Foo", "Bob@foo.org"}})
		h.SetSubject("[PATCH] tests")

		s := ss.Get(STYLE_MSGLIST_DEFAULT, &h)
		if s.Foreground != colorNames["coral"] {
			t.Errorf("expected:#%x got:#%x", colorNames["coral"], s.Foreground)
		}
	})

	t.Run("multiple headers", func(t *testing.T) {
		var h mail.Header
		h.SetAddressList("From", []*mail.Address{{"Bob Foo", "Bob@foo.org"}})
		h.SetSubject("[PATCH] tests")
		h.SetText("X-Clacks-Overhead", "GNU Terry Pratchett")

		s := ss.Get(STYLE_MSGLIST_DEFAULT, &h)
		if s.Foreground != colorNames["pink"] {
			t.Errorf("expected:#%x got:#%x", colorNames["pink"], s.Foreground)
		}
	})

	t.Run("preserves order-sensitivity", func(t *testing.T) {
		var h mail.Header
		h.SetAddressList("From", []*mail.Address{{"Bob Foo", "Bob@foo.org"}})
		h.SetSubject("[PATCH] tests")
		h.SetText("X-Clacks-Overhead", "GNU Terry Pratchett")
		h.SetText("X-Baz", "exact")

		s := ss.Get(STYLE_MSGLIST_DEFAULT, &h)

		// The "pink" entry comes later, so will overrule the more exact
		// match with color "plum"
		if s.Foreground != colorNames["pink"] {
			t.Errorf("expected:#%x got:#%x", colorNames["pink"], s.Foreground)
		}
	})
}
