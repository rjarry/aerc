package ui

const (
	Valid          = 0
	InvalidateTabs = 1 << iota
	InvalidateSidebar
	InvalidateStatusBar
)

const (
	InvalidateAll = InvalidateTabs | InvalidateSidebar | InvalidateStatusBar
)

type Geometry struct {
	row    int
	col    int
	width  int
	height int
}

type AercTab interface {
	Name() string
	Invalid() bool
	Render(at Geometry)
}

type UIState struct {
	Exit         bool
	InvalidPanes uint

	Panes struct {
		TabList   Geometry
		TabView   Geometry
		Sidebar   Geometry
		StatusBar Geometry
	}

	Tabs        []AercTab
	SelectedTab int

	Prompt struct {
		Prompt *string
		Text   *string
		Index  int
		Scroll int
	}
}
