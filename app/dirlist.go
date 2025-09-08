package app

import (
	"bytes"
	"context"
	"math"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/vaxis"
)

type DirectoryLister interface {
	ui.Drawable

	Selected() string
	Previous() string

	Select(string)
	Open(string, string, time.Duration, func(types.WorkerMessage), bool)

	Update(types.WorkerMessage)
	List() []string
	ClearList()

	OnVirtualNode(func())

	NextPrev(int, bool)

	CollapseFolder(string)
	ExpandFolder(string)

	SelectedMsgStore() (*lib.MessageStore, bool)
	MsgStore(string) (*lib.MessageStore, bool)
	SelectedDirectory() *models.Directory
	Directory(string) *models.Directory
	SetMsgStore(*models.Directory, *lib.MessageStore)

	FilterDirs([]string, []string, bool) []string
	GetRUECount(string) (int, int, int)

	UiConfig(string) *config.UIConfig
}

type DirectoryList struct {
	Scrollable
	acctConf  *config.AccountConfig
	store     *lib.DirStore
	dirs      []string
	selecting string
	selected  string
	previous  string
	spinner   *Spinner
	worker    *types.Worker
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewDirectoryList(acctConf *config.AccountConfig,
	worker *types.Worker,
) DirectoryLister {
	dirlist := &DirectoryList{
		acctConf: acctConf,
		store:    lib.NewDirStore(),
		worker:   worker,
	}
	dirlist.NewContext()
	uiConf := dirlist.UiConfig("")
	dirlist.spinner = NewSpinner(uiConf)
	dirlist.spinner.Start()

	if uiConf.DirListTree {
		return NewDirectoryTree(dirlist)
	}

	return dirlist
}

func (dirlist *DirectoryList) NewContext() {
	if dirlist.cancel != nil {
		dirlist.cancel()
	}
	dirlist.ctx, dirlist.cancel = context.WithCancel(context.Background())
}

func (dirlist *DirectoryList) UiConfig(dir string) *config.UIConfig {
	if dir == "" {
		dir = dirlist.Selected()
	}
	return config.Ui().ForAccount(dirlist.acctConf.Name).ForFolder(dir)
}

func (dirlist *DirectoryList) List() []string {
	return dirlist.dirs
}

func (dirlist *DirectoryList) ClearList() {
	dirlist.store = lib.NewDirStore()
	dirlist.dirs = []string{}
}

func (dirlist *DirectoryList) OnVirtualNode(_ func()) {
}

func (dirlist *DirectoryList) Update(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.Done:
		switch msg := msg.InResponseTo().(type) {
		case *types.OpenDirectory:
			dirlist.previous = dirlist.selected
			dirlist.selected = msg.Directory
			dirlist.filterDirsByFoldersConfig()
			hasSelected := slices.Contains(dirlist.dirs, dirlist.selected)
			if !hasSelected && dirlist.selected != "" {
				dirlist.dirs = append(dirlist.dirs, dirlist.selected)
			}
			if dirlist.acctConf.EnableFoldersSort {
				sort.Strings(dirlist.dirs)
			}
			dirlist.sortDirsByFoldersSortConfig()
			store, ok := dirlist.SelectedMsgStore()
			if !ok {
				return
			}
			store.SetContext(msg.Context)
		case *types.ListDirectories:
			dirlist.filterDirsByFoldersConfig()
			dirlist.sortDirsByFoldersSortConfig()
			dirlist.spinner.Stop()
			dirlist.Invalidate()
		case *types.RemoveDirectory:
			dirlist.store.Remove(msg.Directory)
			dirlist.filterDirsByFoldersConfig()
			dirlist.sortDirsByFoldersSortConfig()
		case *types.CreateDirectory:
			dirlist.filterDirsByFoldersConfig()
			dirlist.sortDirsByFoldersSortConfig()
			dirlist.Invalidate()
		}
	case *types.DirectoryInfo:
		dir := dirlist.Directory(msg.Info.Name)
		if dir == nil {
			return
		}
		dir.Exists = msg.Info.Exists
		dir.Recent = msg.Info.Recent
		dir.Unseen = msg.Info.Unseen
		if msg.Refetch {
			store, ok := dirlist.SelectedMsgStore()
			if ok {
				store.Sort(store.GetCurrentSortCriteria(), nil)
			}
		}
	default:
		return
	}
}

func (dirlist *DirectoryList) CollapseFolder(string) {
	// no effect for the DirectoryList
}

func (dirlist *DirectoryList) ExpandFolder(string) {
	// no effect for the DirectoryList
}

func (dirlist *DirectoryList) Select(name string) {
	dirlist.Open(name, "", dirlist.UiConfig(name).DirListDelay, nil, false)
}

func (dirlist *DirectoryList) Open(name string, query string, delay time.Duration,
	cb func(types.WorkerMessage), force bool,
) {
	dirlist.selecting = name

	dirlist.NewContext()

	go func(ctx context.Context) {
		defer log.PanicHandler()

		select {
		case <-time.After(delay):
			dirlist.worker.PostAction(&types.OpenDirectory{
				Context:   ctx,
				Directory: name,
				Query:     query,
				Force:     force,
			},
				func(msg types.WorkerMessage) {
					switch msg := msg.(type) {
					case *types.Error:
						dirlist.selecting = ""
						log.Errorf("(%s) couldn't open directory %s: %v",
							dirlist.acctConf.Name,
							name,
							msg.Error)
					case *types.Cancelled:
						log.Debugf("OpenDirectory cancelled")
					}
					if cb != nil {
						cb(msg)
					}
				})
		case <-ctx.Done():
			log.Tracef("dirlist: skip %s", name)
			return
		}
	}(dirlist.ctx)
}

func (dirlist *DirectoryList) Selected() string {
	return dirlist.selected
}

func (dirlist *DirectoryList) Previous() string {
	return dirlist.previous
}

func (dirlist *DirectoryList) Invalidate() {
	ui.Invalidate()
}

// Returns the Recent, Unread, and Exist counts for the named directory
func (dirlist *DirectoryList) GetRUECount(name string) (int, int, int) {
	dir := dirlist.Directory(name)
	if dir == nil {
		return 0, 0, 0
	}
	return dir.Recent, dir.Unseen, dir.Exists
}

func (dirlist *DirectoryList) Draw(ctx *ui.Context) {
	uiConfig := dirlist.UiConfig("")
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		uiConfig.GetStyle(config.STYLE_DIRLIST_DEFAULT))

	if dirlist.spinner.IsRunning() {
		dirlist.spinner.Draw(ctx)
		return
	}

	if len(dirlist.dirs) == 0 {
		style := uiConfig.GetStyle(config.STYLE_DIRLIST_DEFAULT)
		ctx.Printf(0, 0, style, "%s", uiConfig.EmptyDirlist)
		return
	}

	dirlist.UpdateScroller(ctx.Height(), len(dirlist.dirs))
	dirlist.EnsureScroll(findString(dirlist.dirs, dirlist.selecting))

	textWidth := ctx.Width()
	if dirlist.NeedScrollbar() {
		textWidth -= 1
	}
	if textWidth < 0 {
		return
	}

	listCtx := ctx.Subcontext(0, 0, textWidth, ctx.Height())

	data := state.NewDataSetter()
	data.SetAccount(dirlist.acctConf)

	for i, name := range dirlist.dirs {
		if i < dirlist.Scroll() {
			continue
		}
		row := i - dirlist.Scroll()
		if row >= ctx.Height() {
			break
		}

		data.SetFolder(dirlist.Directory(name))
		data.SetRUE([]string{name}, dirlist.GetRUECount)
		left, right, style := dirlist.renderDir(
			name, uiConfig, data.Data(),
			name == dirlist.selecting, listCtx.Width(),
		)
		listCtx.Printf(0, row, style, "%s %s", left, right)
	}

	if dirlist.NeedScrollbar() {
		scrollBarCtx := ctx.Subcontext(ctx.Width()-1, 0, 1, ctx.Height())
		dirlist.drawScrollbar(scrollBarCtx)
	}
}

func (dirlist *DirectoryList) renderDir(
	path string, conf *config.UIConfig, data models.TemplateData,
	selected bool, width int,
) (string, string, vaxis.Style) {
	var left, right string
	var buf bytes.Buffer

	var styles []config.StyleObject
	var style vaxis.Style

	r, u, _ := dirlist.GetRUECount(path)
	if u > 0 {
		styles = append(styles, config.STYLE_DIRLIST_UNREAD)
	}
	if r > 0 {
		styles = append(styles, config.STYLE_DIRLIST_RECENT)
	}
	conf = conf.ForFolder(path)
	if selected {
		style = conf.GetComposedStyleSelected(
			config.STYLE_DIRLIST_DEFAULT, styles)
	} else {
		style = conf.GetComposedStyle(
			config.STYLE_DIRLIST_DEFAULT, styles)
	}

	err := templates.Render(conf.DirListLeft, &buf, data)
	if err != nil {
		log.Errorf("dirlist-left: %s", err)
		left = err.Error()
		style = conf.GetStyle(config.STYLE_ERROR)
	} else {
		left = buf.String()
	}
	buf.Reset()
	err = templates.Render(conf.DirListRight, &buf, data)
	if err != nil {
		log.Errorf("dirlist-right: %s", err)
		right = err.Error()
		style = conf.GetStyle(config.STYLE_ERROR)
	} else {
		right = buf.String()
	}
	buf.Reset()

	lbuf := ui.StyledString(left)
	ui.ApplyAttrs(lbuf, style)
	lwidth := lbuf.Len()
	rbuf := ui.StyledString(right)
	ui.ApplyAttrs(rbuf, style)
	rwidth := rbuf.Len()

	if lwidth+rwidth+1 > width {
		if rwidth > 3*width/4 {
			rwidth = 3 * width / 4
		}
		lwidth = width - rwidth - 1
		ui.TruncateHead(rbuf, rwidth)
		right = rbuf.Encode()
		ui.Truncate(lbuf, lwidth)
		left = lbuf.Encode()
	} else {
		for i := 0; i < (width - lwidth - rwidth - 1); i += 1 {
			lbuf.Cells = append(lbuf.Cells, vaxis.Cell{
				Character: vaxis.Character{
					Grapheme: " ",
					Width:    1,
				},
			})
		}
		left = lbuf.Encode()
		right = rbuf.Encode()
	}

	return left, right, style
}

func (dirlist *DirectoryList) drawScrollbar(ctx *ui.Context) {
	gutterStyle := vaxis.Style{}
	pillStyle := vaxis.Style{Attribute: vaxis.AttrReverse}

	// gutter
	ctx.Fill(0, 0, 1, ctx.Height(), ' ', gutterStyle)

	// pill
	pillSize := int(math.Ceil(float64(ctx.Height()) * dirlist.PercentVisible()))
	pillOffset := int(math.Floor(float64(ctx.Height()) * dirlist.PercentScrolled()))
	ctx.Fill(0, pillOffset, 1, pillSize, ' ', pillStyle)
}

func (dirlist *DirectoryList) MouseEvent(localX int, localY int, event vaxis.Event) {
	if event, ok := event.(vaxis.Mouse); ok {
		switch event.Button {
		case vaxis.MouseLeftButton:
			clickedDir, ok := dirlist.Clicked(localX, localY)
			if ok {
				dirlist.Select(clickedDir)
			}
		case vaxis.MouseWheelDown:
			dirlist.Next()
		case vaxis.MouseWheelUp:
			dirlist.Prev()
		}
	}
}

func (dirlist *DirectoryList) Clicked(x int, y int) (string, bool) {
	if len(dirlist.dirs) == 0 {
		return "", false
	}
	for i, name := range dirlist.dirs {
		if i == y {
			return name, true
		}
	}
	return "", false
}

func (dirlist *DirectoryList) NextPrevDelta(delta int) {
	if delta == 0 {
		return
	}
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

func (dirlist *DirectoryList) NextPrev(delta int, unseen bool) {
	if unseen {
		ndirs := len(dirlist.dirs)
		for range ndirs {
			dirlist.NextPrevDelta(delta)
			if findString(dirlist.dirs, dirlist.selecting) >= 0 {
				if dirlist.Directory(dirlist.selecting).Unseen > 0 {
					return
				}
			}
		}
	} else {
		dirlist.NextPrevDelta(delta)
	}
}

func (dirlist *DirectoryList) Next() {
	dirlist.NextPrevDelta(1)
}

func (dirlist *DirectoryList) Prev() {
	dirlist.NextPrevDelta(-1)
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
		iInFoldersSort := findFirstMatchingString(foldersSort, dirlist.dirs[i])
		jInFoldersSort := findFirstMatchingString(foldersSort, dirlist.dirs[j])
		if iInFoldersSort != jInFoldersSort {
			if iInFoldersSort >= 0 && jInFoldersSort >= 0 {
				return iInFoldersSort < jInFoldersSort
			}
			if iInFoldersSort >= 0 {
				return true
			}
			if jInFoldersSort >= 0 {
				return false
			}
		}
		return dirlist.dirs[i] < dirlist.dirs[j]
	})
}

// filterDirsByFoldersConfig sets dirlist.dirs to the filtered subset of the
// dirstore, based on AccountConfig.Folders (inclusion) and
// AccountConfig.FoldersExclude (exclusion), in that order.
func (dirlist *DirectoryList) filterDirsByFoldersConfig() {
	dirlist.dirs = dirlist.store.List()

	// 'folders' (if available) is used to make the initial list and
	// 'folders-exclude' removes from that list.
	configFolders := dirlist.acctConf.Folders
	dirlist.dirs = dirlist.FilterDirs(dirlist.dirs, configFolders, false)

	configFoldersExclude := dirlist.acctConf.FoldersExclude
	dirlist.dirs = dirlist.FilterDirs(dirlist.dirs, configFoldersExclude, true)
}

// FilterDirs filters directories by the supplied filter. If exclude is false,
// the filter will only include directories from orig which exist in filters.
// If exclude is true, the directories in filters are removed from orig
func (dirlist *DirectoryList) FilterDirs(orig, filters []string, exclude bool) []string {
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

func (dirlist *DirectoryList) SelectedMsgStore() (*lib.MessageStore, bool) {
	return dirlist.store.MessageStore(dirlist.selected)
}

func (dirlist *DirectoryList) MsgStore(name string) (*lib.MessageStore, bool) {
	return dirlist.store.MessageStore(name)
}

func (dirlist *DirectoryList) SelectedDirectory() *models.Directory {
	return dirlist.store.Directory(dirlist.selected)
}

func (dirlist *DirectoryList) Directory(name string) *models.Directory {
	return dirlist.store.Directory(name)
}

func (dirlist *DirectoryList) SetMsgStore(dir *models.Directory, msgStore *lib.MessageStore) {
	dirlist.store.SetMessageStore(dir, msgStore)
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

func findFirstMatchingString(slice []string, str string) int {
	for i, s := range slice {
		matches, err := filepath.Match(s, str)
		if err == nil && matches {
			return i
		}
	}
	return -1
}
