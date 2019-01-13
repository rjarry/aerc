package widgets

import (
	"log"
	"sort"
	"strings"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type DirectoryList struct {
	conf         *config.AccountConfig
	dirs         *ui.List
	logger       *log.Logger
	onInvalidate func(d ui.Drawable)
	worker       *types.Worker
}

func NewDirectoryList(conf *config.AccountConfig,
	logger *log.Logger, worker *types.Worker) *DirectoryList {

	return &DirectoryList{
		conf:   conf,
		dirs:   ui.NewList(),
		logger: logger,
		worker: worker,
	}
}

func (dirlist *DirectoryList) UpdateList() {
	var dirs []ui.Drawable
	dirlist.worker.PostAction(
		&types.ListDirectories{}, func(msg types.WorkerMessage) {

			switch msg := msg.(type) {
			case *types.Directory:
				if len(dirlist.conf.Folders) > 1 {
					idx := sort.SearchStrings(dirlist.conf.Folders, msg.Name)
					if idx == len(dirlist.conf.Folders) ||
						dirlist.conf.Folders[idx] != msg.Name {
						break
					}
				}
				dirs = append(dirs, directoryEntry(msg.Name))
			case *types.Done:
				sort.Slice(dirs, func(_a, _b int) bool {
					a, _ := dirs[_a].(directoryEntry)
					b, _ := dirs[_b].(directoryEntry)
					return strings.Compare(string(a), string(b)) > 0
				})
				dirlist.dirs.Set(dirs)
				dirlist.dirs.Select(0)
			}
		})
}

func (dirlist *DirectoryList) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	dirlist.dirs.OnInvalidate(func(_ ui.Drawable) {
		onInvalidate(dirlist)
	})
}

func (dirlist *DirectoryList) Invalidate() {
	dirlist.dirs.Invalidate()
}

func (dirlist *DirectoryList) Draw(ctx *ui.Context) {
	dirlist.dirs.Draw(ctx)
}

type directoryEntry string

func (d directoryEntry) OnInvalidate(_ func(_ ui.Drawable)) {
}

func (d directoryEntry) Invalidate() {
}

func (d directoryEntry) Draw(ctx *ui.Context) {
	d.DrawWithSelected(ctx, false)
}

func (d directoryEntry) DrawWithSelected(ctx *ui.Context, selected bool) {
	style := tcell.StyleDefault
	if selected {
		style = style.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	}
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
	ctx.Printf(0, 0, style, "%s", d)
}
