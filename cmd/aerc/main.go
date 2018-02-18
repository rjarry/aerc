package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/ui"
)

type fill rune

func (f fill) Draw(ctx *ui.Context) {
	for x := 0; x < ctx.Width(); x += 1 {
		for y := 0; y < ctx.Height(); y += 1 {
			ctx.SetCell(x, y, rune(f), tb.ColorDefault, tb.ColorDefault)
		}
	}
}

func (f fill) OnInvalidate(callback func(d ui.Drawable)) {
	// no-op
}

func (f fill) Invalidate() {
	// no-op
}

func main() {
	var logOut io.Writer
	var logger *log.Logger
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logOut = os.Stdout
	} else {
		logOut = ioutil.Discard
	}
	logger = log.New(logOut, "", log.LstdFlags)
	logger.Println("Starting up aerc")

	conf, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}

	tabs := ui.NewTabs()
	tabs.Add(fill('★'), "白い星")
	tabs.Add(fill('☆'), "empty stars")

	grid := ui.NewGrid()
	grid.Rows = []ui.DimSpec{
		ui.DimSpec{ui.SIZE_EXACT, 1},
		ui.DimSpec{ui.SIZE_WEIGHT, 1},
		ui.DimSpec{ui.SIZE_WEIGHT, 1},
		ui.DimSpec{ui.SIZE_EXACT, 1},
	}
	grid.Columns = []ui.DimSpec{
		ui.DimSpec{ui.SIZE_WEIGHT, 3},
		ui.DimSpec{ui.SIZE_WEIGHT, 2},
	}
	grid.AddChild(tabs.TabStrip).At(0, 0).Span(1, 2)
	grid.AddChild(tabs.TabContent).At(1, 0).Span(1, 2)
	grid.AddChild(fill('.')).At(2, 0).Span(1, 2)
	grid.AddChild(fill('•')).At(2, 1).Span(1, 1)
	grid.AddChild(fill('+')).At(3, 0).Span(1, 2)

	_ui, err := ui.Initialize(conf, grid)
	if err != nil {
		panic(err)
	}
	defer _ui.Close()

	go (func() {
		for {
			time.Sleep(1 * time.Second)
			tabs.Select((tabs.Selected + 1) % 2)
		}
	})()

	for !_ui.Exit {
		if !_ui.Tick() {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
