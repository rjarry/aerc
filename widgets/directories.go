package widgets

import (
	"log"
	"sort"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type DirectoryList struct {
	dirs         []string
	logger       *log.Logger
	onInvalidate func(d ui.Drawable)
	worker       *types.Worker
}

func NewDirectoryList(logger *log.Logger, worker *types.Worker) *DirectoryList {
	return &DirectoryList{logger: logger, worker: worker}
}

func (dirlist *DirectoryList) UpdateList() {
	var dirs []string
	dirlist.worker.PostAction(
		&types.ListDirectories{}, func(msg types.WorkerMessage) {

			switch msg := msg.(type) {
			case *types.Directory:
				dirs = append(dirs, msg.Name)
			case *types.Done:
				sort.Strings(dirs)
				dirlist.dirs = dirs
				dirlist.Invalidate()
			}
		})
}

func (dirlist *DirectoryList) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	dirlist.onInvalidate = onInvalidate
}

func (dirlist *DirectoryList) Invalidate() {
	if dirlist.onInvalidate != nil {
		dirlist.onInvalidate(dirlist)
	}
}

func (dirlist *DirectoryList) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	for i, name := range dirlist.dirs {
		if i >= ctx.Height() {
			break
		}
		ctx.Printf(0, i, tcell.StyleDefault, "%s", name)
	}
}
