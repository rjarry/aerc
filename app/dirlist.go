package app

import (
	"bytes"
	"context"
	"math"
	"regexp"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type DirectoryLister interface {
	ui.Drawable

	Selected() string
	Select(string)
	Open(string, time.Duration, func(types.WorkerMessage))

	Update(types.WorkerMessage)
	List() []string
	ClearList()

	OnVirtualNode(func())

	NextPrev(int)

	CollapseFolder()
	ExpandFolder()

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
	spinner   *Spinner
	worker    *types.Worker
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewDirectoryList(acctConf *config.AccountConfig,
	worker *types.Worker,
) DirectoryLister {
	ctx, cancel := context.WithCancel(context.Background())

	dirlist := &DirectoryList{
		acctConf: acctConf,
		store:    lib.NewDirStore(),
		worker:   worker,
		ctx:      ctx,
		cancel:   cancel,
	}
	uiConf := dirlist.UiConfig("")
	dirlist.spinner = NewSpinner(uiConf)
	dirlist.spinner.Start()

	if uiConf.DirListTree {
		return NewDirectoryTree(dirlist)
	}

	return dirlist
}

func (dirlist *DirectoryList) UiConfig(dir string) *config.UIConfig {
	if dir == "" {
		dir = dirlist.Selected()
	}
	return config.Ui.ForAccount(dirlist.acctConf.Name).ForFolder(dir)
}

func (dirlist *DirectoryList) List() []string {
	return dirlist.dirs
}

func (dirlist *DirectoryList) ClearList() {
	dirlist.dirs = []string{}
}

func (dirlist *DirectoryList) OnVirtualNode(_ func()) {
}

func (dirlist *DirectoryList) Update(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.Done:
		switch msg := msg.InResponseTo().(type) {
		case *types.OpenDirectory:
			dirlist.selected = msg.Directory
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

func (dirlist *DirectoryList) CollapseFolder() {
	// no effect for the DirectoryList
}

func (dirlist *DirectoryList) ExpandFolder() {
	// no effect for the DirectoryList
}

func (dirlist *DirectoryList) Select(name string) {
	dirlist.Open(name, dirlist.UiConfig(name).DirListDelay, nil)
}

func (dirlist *DirectoryList) Open(name string, delay time.Duration,
	cb func(types.WorkerMessage),
) {
	dirlist.selecting = name

	dirlist.cancel()
	dirlist.ctx, dirlist.cancel = context.WithCancel(context.Background())

	go func(ctx context.Context) {
		defer log.PanicHandler()

		select {
		case <-time.After(delay):
			dirlist.worker.PostAction(&types.OpenDirectory{
				Context:   ctx,
				Directory: name,
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
		ctx.Printf(0, 0, style, uiConfig.EmptyDirlist)
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
) (string, string, tcell.Style) {
	var left, right string
	var buf bytes.Buffer

	var styles []config.StyleObject
	var style tcell.Style

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

	lbuf := parse.ParseANSI(left)
	lbuf.ApplyAttrs(style)
	lwidth := lbuf.Len()
	rbuf := parse.ParseANSI(right)
	rbuf.ApplyAttrs(style)
	rwidth := rbuf.Len()

	if lwidth+rwidth+1 > width {
		if rwidth > 3*width/4 {
			rwidth = 3 * width / 4
		}
		lwidth = width - rwidth - 1
		right = rbuf.TruncateHead(rwidth, '…')
		left = lbuf.Truncate(lwidth-1, '…')
	} else {
		for i := 0; i < (width - lwidth - rwidth - 1); i += 1 {
			lbuf.Write(' ', tcell.StyleDefault)
		}
		left = lbuf.String()
		right = rbuf.String()
	}

	return left, right, style
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
	if event, ok := event.(*tcell.EventMouse); ok {
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
