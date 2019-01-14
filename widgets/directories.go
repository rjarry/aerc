package widgets

import (
	"log"
	"sort"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type DirectoryList struct {
	conf         *config.AccountConfig
	dirs         []string
	logger       *log.Logger
	onInvalidate func(d ui.Drawable)
	selected     string
	spinner      *Spinner
	worker       *types.Worker
}

func NewDirectoryList(conf *config.AccountConfig,
	logger *log.Logger, worker *types.Worker) *DirectoryList {

	dirlist := &DirectoryList{
		conf:    conf,
		logger:  logger,
		spinner: NewSpinner(),
		worker:  worker,
	}
	dirlist.spinner.OnInvalidate(func(_ ui.Drawable) {
		dirlist.Invalidate()
	})
	dirlist.spinner.Start()
	return dirlist
}

func (dirlist *DirectoryList) UpdateList(done func(dirs []string)) {
	var dirs []string
	dirlist.worker.PostAction(
		&types.ListDirectories{}, func(msg types.WorkerMessage) {

			switch msg := msg.(type) {
			case *types.Directory:
				dirs = append(dirs, msg.Name)
			case *types.Done:
				sort.Strings(dirs)
				dirlist.dirs = dirs
				dirlist.spinner.Stop()
				dirlist.Invalidate()
				if done != nil {
					done(dirs)
				}
			}
		})
}

func (dirlist *DirectoryList) Select(name string) {
	dirlist.selected = name
	dirlist.worker.PostAction(&types.OpenDirectory{Directory: name}, nil)
	dirlist.Invalidate()
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

	if dirlist.spinner.IsRunning() {
		dirlist.spinner.Draw(ctx)
		return
	}

	row := 0
	for _, name := range dirlist.dirs {
		if row >= ctx.Height() {
			break
		}
		if len(dirlist.conf.Folders) > 1 && name != dirlist.selected {
			idx := sort.SearchStrings(dirlist.conf.Folders, name)
			if idx == len(dirlist.conf.Folders) ||
				dirlist.conf.Folders[idx] != name {
				continue
			}
		}
		style := tcell.StyleDefault
		if name == dirlist.selected {
			style = style.Background(tcell.ColorWhite).
				Foreground(tcell.ColorBlack)
		}
		ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
		ctx.Printf(0, row, style, "%s", name)
		row++
	}
}
