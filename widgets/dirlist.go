package widgets

import (
	"log"
	"regexp"
	"sort"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type DirectoryList struct {
	ui.Invalidatable
	acctConf  *config.AccountConfig
	uiConf    *config.UIConfig
	store     *lib.DirStore
	dirs      []string
	logger    *log.Logger
	selecting string
	selected  string
	spinner   *Spinner
	worker    *types.Worker
}

func NewDirectoryList(acctConf *config.AccountConfig, uiConf *config.UIConfig,
	logger *log.Logger, worker *types.Worker) *DirectoryList {

	dirlist := &DirectoryList{
		acctConf: acctConf,
		uiConf:   uiConf,
		logger:   logger,
		spinner:  NewSpinner(uiConf),
		store:    lib.NewDirStore(),
		worker:   worker,
	}
	dirlist.spinner.OnInvalidate(func(_ ui.Drawable) {
		dirlist.Invalidate()
	})
	dirlist.spinner.Start()
	return dirlist
}

func (dirlist *DirectoryList) List() []string {
	return dirlist.store.List()
}

func (dirlist *DirectoryList) UpdateList(done func(dirs []string)) {
	// TODO: move this logic into dirstore
	var dirs []string
	dirlist.worker.PostAction(
		&types.ListDirectories{}, func(msg types.WorkerMessage) {

			switch msg := msg.(type) {
			case *types.Directory:
				dirs = append(dirs, msg.Dir.Name)
			case *types.Done:
				sort.Strings(dirs)
				dirlist.store.Update(dirs)
				dirlist.filterDirsByFoldersConfig()
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
				dirlist.filterDirsByFoldersConfig()
				hasSelected := false
				for _, d := range dirlist.dirs {
					if d == dirlist.selected {
						hasSelected = true
						break
					}
				}
				if !hasSelected && dirlist.selected != "" {
					dirlist.dirs = append(dirlist.dirs, dirlist.selected)
				}
				sort.Strings(dirlist.dirs)
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

	if len(dirlist.dirs) == 0 {
		style := tcell.StyleDefault
		ctx.Printf(0, 0, style, dirlist.uiConf.EmptyDirlist)
		return
	}

	row := 0
	for _, name := range dirlist.dirs {
		if row >= ctx.Height() {
			break
		}
		style := tcell.StyleDefault
		if name == dirlist.selected {
			style = style.Reverse(true)
		} else if name == dirlist.selecting {
			style = style.Reverse(true)
			style = style.Foreground(tcell.ColorGray)
		}
		ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
		ctx.Printf(0, row, style, "%s", name)
		row++
	}
}

func (dirlist *DirectoryList) MouseEvent(localX int, localY int, event tcell.Event) {
	switch event := event.(type) {
	case *tcell.EventMouse:
		switch event.Buttons() {
		case tcell.Button1:
			clickedDir, ok := dirlist.Clicked(localX, localY)
			if ok {
				dirlist.Select(clickedDir)
			}
		case tcell.WheelDown:
			dirlist.Next()
		case tcell.WheelUp:
			dirlist.Prev()
		}
	}
}

func (dirlist *DirectoryList) Clicked(x int, y int) (string, bool) {
	if dirlist.dirs == nil || len(dirlist.dirs) == 0 {
		return "", false
	}
	for i, name := range dirlist.dirs {
		if i == y {
			return name, true
		}
	}
	return "", false
}

func (dirlist *DirectoryList) NextPrev(delta int) {
	curIdx := sort.SearchStrings(dirlist.dirs, dirlist.selected)
	if curIdx == len(dirlist.dirs) {
		return
	}
	newIdx := curIdx + delta
	ndirs := len(dirlist.dirs)
	if newIdx < 0 {
		newIdx = ndirs - 1
	} else if newIdx >= ndirs {
		newIdx = 0
	}
	dirlist.Select(dirlist.dirs[newIdx])
}

func (dirlist *DirectoryList) Next() {
	dirlist.NextPrev(1)
}

func (dirlist *DirectoryList) Prev() {
	dirlist.NextPrev(-1)
}

func folderMatches(folder string, pattern string) bool {
	if len(pattern) == 0 {
		return false
	}
	if pattern[0] == '~' {
		r, err := regexp.Compile(pattern[1:])
		if err != nil {
			return false
		}
		return r.Match([]byte(folder))
	}
	return pattern == folder
}

// filterDirsByFoldersConfig sets dirlist.dirs to the filtered subset of the
// dirstore, based on the AccountConfig.Folders option
func (dirlist *DirectoryList) filterDirsByFoldersConfig() {
	dirlist.dirs = dirlist.store.List()
	// config option defaults to show all if unset
	if len(dirlist.acctConf.Folders) == 0 {
		return
	}
	var filtered []string
	for _, folder := range dirlist.dirs {
		for _, cfgfolder := range dirlist.acctConf.Folders {
			if folderMatches(folder, cfgfolder) {
				filtered = append(filtered, folder)
				break
			}
		}
	}
	dirlist.dirs = filtered
}

func (dirlist *DirectoryList) SelectedMsgStore() (*lib.MessageStore, bool) {
	return dirlist.store.MessageStore(dirlist.selected)
}

func (dirlist *DirectoryList) MsgStore(name string) (*lib.MessageStore, bool) {
	return dirlist.store.MessageStore(name)
}

func (dirlist *DirectoryList) SetMsgStore(name string, msgStore *lib.MessageStore) {
	dirlist.store.SetMessageStore(name, msgStore)
}
