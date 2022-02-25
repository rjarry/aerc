package widgets

import (
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	libsort "git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type DirectoryLister interface {
	ui.Drawable

	Selected() string
	Select(string)

	UpdateList(func([]string))
	List() []string

	NextPrev(int)

	CollapseFolder()
	ExpandFolder()

	SelectedMsgStore() (*lib.MessageStore, bool)
	MsgStore(string) (*lib.MessageStore, bool)
	SetMsgStore(string, *lib.MessageStore)
}

type DirectoryList struct {
	ui.Invalidatable
	Scrollable
	aercConf   *config.AercConfig
	acctConf   *config.AccountConfig
	store      *lib.DirStore
	dirs       []string
	logger     *log.Logger
	selecting  string
	selected   string
	spinner    *Spinner
	worker     *types.Worker
	skipSelect chan bool
}

func NewDirectoryList(conf *config.AercConfig, acctConf *config.AccountConfig,
	logger *log.Logger, worker *types.Worker) DirectoryLister {

	dirlist := &DirectoryList{
		aercConf:   conf,
		acctConf:   acctConf,
		logger:     logger,
		store:      lib.NewDirStore(),
		worker:     worker,
		skipSelect: make(chan bool),
	}
	uiConf := dirlist.UiConfig()
	dirlist.spinner = NewSpinner(&uiConf)
	dirlist.spinner.OnInvalidate(func(_ ui.Drawable) {
		dirlist.Invalidate()
	})
	dirlist.spinner.Start()

	if uiConf.DirListTree {
		return NewDirectoryTree(dirlist, string(os.PathSeparator))
	}

	return dirlist
}

func (dirlist *DirectoryList) UiConfig() config.UIConfig {
	return dirlist.aercConf.GetUiConfig(map[config.ContextType]string{
		config.UI_CONTEXT_ACCOUNT: dirlist.acctConf.Name,
		config.UI_CONTEXT_FOLDER:  dirlist.Selected(),
	})
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
				dirlist.store.Update(dirs)
				dirlist.filterDirsByFoldersConfig()
				dirlist.sortDirsByFoldersSortConfig()
				dirlist.store.Update(dirlist.dirs)
				dirlist.spinner.Stop()
				dirlist.Invalidate()
				if done != nil {
					done(dirs)
				}
			}
		})
}

func (dirlist *DirectoryList) CollapseFolder() {
	// no effect for the DirectoryList
}

func (dirlist *DirectoryList) ExpandFolder() {
	// no effect for the DirectoryList
}

func (dirlist *DirectoryList) Select(name string) {
	dirlist.selecting = name

	close(dirlist.skipSelect)
	dirlist.skipSelect = make(chan bool)

	go func() {
		select {
		case <-time.After(dirlist.UiConfig().DirListDelay):
			dirlist.worker.PostAction(&types.OpenDirectory{Directory: name},
				func(msg types.WorkerMessage) {
					switch msg.(type) {
					case *types.Error:
						dirlist.selecting = ""
						dirlist.selected = ""
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
						if dirlist.acctConf.EnableFoldersSort {
							sort.Strings(dirlist.dirs)
						}
						dirlist.sortDirsByFoldersSortConfig()
					}
					dirlist.Invalidate()
				})
			dirlist.Invalidate()
		case <-dirlist.skipSelect:
			dirlist.logger.Println("dirlist: skip", name)
			return
		}
	}()
}

func (dirlist *DirectoryList) Selected() string {
	return dirlist.selected
}

func (dirlist *DirectoryList) Invalidate() {
	dirlist.DoInvalidate(dirlist)
}

func (dirlist *DirectoryList) getDirString(name string, width int, recentUnseen func() string) string {
	percent := false
	rightJustify := false
	formatted := ""
	doRightJustify := func(s string) {
		formatted = runewidth.FillRight(formatted, width-len(s))
		formatted = runewidth.Truncate(formatted, width-len(s), "…")
	}
	for _, char := range dirlist.UiConfig().DirListFormat {
		switch char {
		case '%':
			if percent {
				formatted += string(char)
				percent = false
			} else {
				percent = true
			}
		case '>':
			if percent {
				rightJustify = true
			}
		case 'n':
			if percent {
				if rightJustify {
					doRightJustify(name)
					rightJustify = false
				}
				formatted += name
				percent = false
			}
		case 'r':
			if percent {
				rString := recentUnseen()
				if rightJustify {
					doRightJustify(rString)
					rightJustify = false
				}
				formatted += rString
				percent = false
			}
		default:
			formatted += string(char)
		}
	}
	return formatted
}

func (dirlist *DirectoryList) getRUEString(name string) string {
	msgStore, ok := dirlist.MsgStore(name)
	if !ok {
		return ""
	}
	var totalRecent, totalUnseen, totalExists int
	if msgStore.DirInfo.AccurateCounts {
		totalRecent = msgStore.DirInfo.Recent
		totalUnseen = msgStore.DirInfo.Unseen
		totalExists = msgStore.DirInfo.Exists
	} else {
		totalRecent, totalUnseen = countRUE(msgStore)
		// use the total count from the dirinfo, else we only count already
		// fetched messages
		totalExists = msgStore.DirInfo.Exists
	}
	rueString := ""
	if totalRecent > 0 {
		rueString = fmt.Sprintf("%d/%d/%d", totalRecent, totalUnseen, totalExists)
	} else if totalUnseen > 0 {
		rueString = fmt.Sprintf("%d/%d", totalUnseen, totalExists)
	} else if totalExists > 0 {
		rueString = fmt.Sprintf("%d", totalExists)
	}
	return rueString
}

func (dirlist *DirectoryList) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		dirlist.UiConfig().GetStyle(config.STYLE_DIRLIST_DEFAULT))

	if dirlist.spinner.IsRunning() {
		dirlist.spinner.Draw(ctx)
		return
	}

	if len(dirlist.dirs) == 0 {
		style := dirlist.UiConfig().GetStyle(config.STYLE_DIRLIST_DEFAULT)
		ctx.Printf(0, 0, style, dirlist.UiConfig().EmptyDirlist)
		return
	}

	dirlist.UpdateScroller(ctx.Height(), len(dirlist.dirs))
	dirlist.EnsureScroll(findString(dirlist.dirs, dirlist.selecting))

	textWidth := ctx.Width()
	if dirlist.NeedScrollbar() {
		textWidth -= 1
	}
	if textWidth < 0 {
		textWidth = 0
	}

	for i, name := range dirlist.dirs {
		if i < dirlist.Scroll() {
			continue
		}
		row := i - dirlist.Scroll()
		if row >= ctx.Height() {
			break
		}

		style := dirlist.UiConfig().GetStyle(config.STYLE_DIRLIST_DEFAULT)
		if name == dirlist.selecting {
			style = dirlist.UiConfig().GetStyleSelected(config.STYLE_DIRLIST_DEFAULT)
		}
		ctx.Fill(0, row, textWidth, 1, ' ', style)

		dirString := dirlist.getDirString(name, textWidth, func() string {
			return dirlist.getRUEString(name)
		})

		ctx.Printf(0, row, style, dirString)
	}

	if dirlist.NeedScrollbar() {
		scrollBarCtx := ctx.Subcontext(ctx.Width()-1, 0, 1, ctx.Height())
		dirlist.drawScrollbar(scrollBarCtx)
	}
}

func (dirlist *DirectoryList) drawScrollbar(ctx *ui.Context) {
	gutterStyle := tcell.StyleDefault
	pillStyle := tcell.StyleDefault.Reverse(true)

	// gutter
	ctx.Fill(0, 0, 1, ctx.Height(), ' ', gutterStyle)

	// pill
	pillSize := int(math.Ceil(float64(ctx.Height()) * dirlist.PercentVisible()))
	pillOffset := int(math.Floor(float64(ctx.Height()) * dirlist.PercentScrolled()))
	ctx.Fill(0, pillOffset, 1, pillSize, ' ', pillStyle)
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
	curIdx := findString(dirlist.dirs, dirlist.selecting)
	if curIdx == len(dirlist.dirs) {
		return
	}
	newIdx := curIdx + delta
	ndirs := len(dirlist.dirs)

	if ndirs == 0 {
		return
	}

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

// sortDirsByFoldersSortConfig sets dirlist.dirs to be sorted based on the
// AccountConfig.FoldersSort option. Folders not included in the option
// will be appended at the end in alphabetical order
func (dirlist *DirectoryList) sortDirsByFoldersSortConfig() {
	if !dirlist.acctConf.EnableFoldersSort {
		return
	}

	sort.Slice(dirlist.dirs, func(i, j int) bool {
		foldersSort := dirlist.acctConf.FoldersSort
		iInFoldersSort := findString(foldersSort, dirlist.dirs[i])
		jInFoldersSort := findString(foldersSort, dirlist.dirs[j])
		if iInFoldersSort >= 0 && jInFoldersSort >= 0 {
			return iInFoldersSort < jInFoldersSort
		}
		if iInFoldersSort >= 0 {
			return true
		}
		if jInFoldersSort >= 0 {
			return false
		}
		return dirlist.dirs[i] < dirlist.dirs[j]
	})
}

// filterDirsByFoldersConfig sets dirlist.dirs to the filtered subset of the
// dirstore, based on AccountConfig.Folders (inclusion) and
// AccountConfig.FoldersExclude (exclusion), in that order.
func (dirlist *DirectoryList) filterDirsByFoldersConfig() {
	filterDirs := func(orig, filters []string, exclude bool) []string {
		if len(filters) == 0 {
			return orig
		}
		var dest []string
		for _, folder := range orig {
			// When excluding, include things by default, and vice-versa
			include := exclude
			for _, f := range filters {
				if folderMatches(folder, f) {
					// If matched an exclusion, don't include
					// If matched an inclusion, do include
					include = !exclude
					break
				}
			}
			if include {
				dest = append(dest, folder)
			}
		}
		return dest
	}

	dirlist.dirs = dirlist.store.List()

	// 'folders' (if available) is used to make the initial list and
	// 'folders-exclude' removes from that list.
	configFolders := dirlist.acctConf.Folders
	dirlist.dirs = filterDirs(dirlist.dirs, configFolders, false)

	configFoldersExclude := dirlist.acctConf.FoldersExclude
	dirlist.dirs = filterDirs(dirlist.dirs, configFoldersExclude, true)
}

func (dirlist *DirectoryList) SelectedMsgStore() (*lib.MessageStore, bool) {
	return dirlist.store.MessageStore(dirlist.selected)
}

func (dirlist *DirectoryList) MsgStore(name string) (*lib.MessageStore, bool) {
	return dirlist.store.MessageStore(name)
}

func (dirlist *DirectoryList) SetMsgStore(name string, msgStore *lib.MessageStore) {
	dirlist.store.SetMessageStore(name, msgStore)
	msgStore.OnUpdateDirs(func() {
		dirlist.Invalidate()
	})
}

func findString(slice []string, str string) int {
	for i, s := range slice {
		if str == s {
			return i
		}
	}
	return -1
}

func (dirlist *DirectoryList) getSortCriteria() []*types.SortCriterion {
	if len(dirlist.UiConfig().Sort) == 0 {
		return nil
	}
	criteria, err := libsort.GetSortCriteria(dirlist.UiConfig().Sort)
	if err != nil {
		dirlist.logger.Printf("getSortCriteria failed: %v", err)
		return nil
	}
	return criteria
}

func countRUE(msgStore *lib.MessageStore) (recent, unread int) {
	for _, msg := range msgStore.Messages {
		if msg == nil {
			continue
		}
		seen := false
		isrecent := false
		for _, flag := range msg.Flags {
			if flag == models.SeenFlag {
				seen = true
			} else if flag == models.RecentFlag {
				isrecent = true
			}
		}
		if !seen {
			if isrecent {
				recent++
			} else {
				unread++
			}
		}
	}
	return recent, unread
}
