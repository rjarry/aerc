package ui

import (
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

const (
	Valid             = 0
	InvalidateTabList = 1 << iota
	InvalidateTabView
	InvalidateStatusBar
)

const (
	InvalidateAll = InvalidateTabList |
		InvalidateTabView |
		InvalidateStatusBar
)

type Geometry struct {
	Row    int
	Col    int
	Width  int
	Height int
}

type AercTab interface {
	Name() string
	Render(at Geometry)
	SetParent(parent *UIState)
}

type WorkerListener interface {
	GetChannel() chan types.WorkerMessage
	HandleMessage(msg types.WorkerMessage)
}

type wrappedMessage struct {
	msg      types.WorkerMessage
	listener WorkerListener
}

type UIState struct {
	Config       *config.AercConfig
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

	tbEvents chan tb.Event
	// Aggregate channel for all worker messages
	workerEvents chan wrappedMessage
}
