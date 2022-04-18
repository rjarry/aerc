package statusline

import "strings"

type Texter interface {
	Connected() string
	Disconnected() string
	Passthrough() string
	Sorting() string
	Threading() string
	FormatFilter(string) string
	FormatSearch(string) string
}

type text struct{}

func (t text) Connected() string {
	return "Connected"
}

func (t text) Disconnected() string {
	return "Disconnected"
}

func (t text) Passthrough() string {
	return "passthrough"
}

func (t text) Sorting() string {
	return "sorting"
}

func (t text) Threading() string {
	return "threading"
}

func (t text) FormatFilter(s string) string {
	return s
}

func (t text) FormatSearch(s string) string {
	return s
}

type icon struct{}

func (i icon) Connected() string {
	return "✓"
}

func (i icon) Disconnected() string {
	return "✘"
}

func (i icon) Passthrough() string {
	return "➔"
}

func (i icon) Sorting() string {
	return "⚙"
}

func (i icon) Threading() string {
	return "🧵"
}

func (i icon) FormatFilter(s string) string {
	return strings.ReplaceAll(s, "filter", "🔦")
}

func (i icon) FormatSearch(s string) string {
	return strings.ReplaceAll(s, "search", "🔎")
}
