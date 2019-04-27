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
	ui.Invalidatable
	conf         *config.AccountConfig
	dirs         []string
	logger       *log.Logger
	selecting    string
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
	dirlist.selecting = name
	dirlist.worker.PostAction(&types.OpenDirectory{Directory: name},
		func(msg types.WorkerMessage) {
			switch msg.(type) {
			case *types.Error:
				dirlist.selecting = ""
			case *types.Done:
				dirlist.selected = dirlist.selecting
			}
			dirlist.Invalidate()
		})
	dirlist.Invalidate()
}

func (dirlist *DirectoryList) Selected() string {
	return dirlist.selected
}

func (dirlist *DirectoryList) Invalidate() {
	dirlist.DoInvalidate(dirlist)
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
			style = style.Reverse(true)
		}
		ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
		ctx.Printf(0, row, style, "%s", name)
		row++
	}
}

func (dirlist *DirectoryList) nextPrev(delta int) {
	for i, dir := range dirlist.dirs {
		if dir == dirlist.selected {
			var j int
			ndirs := len(dirlist.dirs)
			for j = i + delta; j != i; j += delta {
				if j < 0 {
					j = ndirs - 1
				}
				if j >= ndirs {
					j = 0
				}
				name := dirlist.dirs[j]
				if len(dirlist.conf.Folders) > 1 && name != dirlist.selected {
					idx := sort.SearchStrings(dirlist.conf.Folders, name)
					if idx == len(dirlist.conf.Folders) ||
						dirlist.conf.Folders[idx] != name {

						continue
					}
				}
				break
			}
			dirlist.Select(dirlist.dirs[j])
			break
		}
	}
}

func (dirlist *DirectoryList) Next() {
	dirlist.nextPrev(1)
}

func (dirlist *DirectoryList) Prev() {
	dirlist.nextPrev(-1)
}
