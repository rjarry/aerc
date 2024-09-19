package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/vaxis"
)

type DirectoryTree struct {
	*DirectoryList

	listIdx int
	list    []*types.Thread

	virtual   bool
	virtualCb func()
}

func NewDirectoryTree(dirlist *DirectoryList) DirectoryLister {
	dt := &DirectoryTree{
		DirectoryList: dirlist,
		listIdx:       -1,
		virtualCb:     func() {},
	}
	return dt
}

func (dt *DirectoryTree) OnVirtualNode(cb func()) {
	dt.virtualCb = cb
}

func (dt *DirectoryTree) Selected() string {
	if dt.listIdx < 0 || dt.listIdx >= len(dt.list) {
		return dt.DirectoryList.Selected()
	}
	node := dt.list[dt.listIdx]
	elems := dt.nodeElems(node)
	n := countLevels(node)
	if n < 0 || n >= len(elems) {
		return ""
	}
	return strings.Join(elems[:(n+1)], dt.DirectoryList.worker.PathSeparator())
}

func (dt *DirectoryTree) SelectedDirectory() *models.Directory {
	if dt.virtual {
		return &models.Directory{
			Name: dt.Selected(),
			Role: models.VirtualRole,
		}
	}
	return dt.DirectoryList.SelectedDirectory()
}

func (dt *DirectoryTree) ClearList() {
	dt.list = make([]*types.Thread, 0)
}

func (dt *DirectoryTree) Update(msg types.WorkerMessage) {
	selected := dt.Selected()
	switch msg := msg.(type) {
	case *types.Done:
		switch msg.InResponseTo().(type) {
		case *types.RemoveDirectory, *types.ListDirectories, *types.CreateDirectory:
			dt.DirectoryList.Update(msg)
			dt.buildTree()
			if selected != "" {
				dt.reindex(selected)
			}
			dt.Invalidate()
		default:
			dt.DirectoryList.Update(msg)
		}
	default:
		dt.DirectoryList.Update(msg)
	}
}

func (dt *DirectoryTree) Draw(ctx *ui.Context) {
	uiConfig := dt.UiConfig("")
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		uiConfig.GetStyle(config.STYLE_DIRLIST_DEFAULT))

	if dt.DirectoryList.spinner.IsRunning() {
		dt.DirectoryList.spinner.Draw(ctx)
		return
	}

	n := dt.countVisible(dt.list)
	if n == 0 || dt.listIdx < 0 {
		style := uiConfig.GetStyle(config.STYLE_DIRLIST_DEFAULT)
		ctx.Printf(0, 0, style, "%s", uiConfig.EmptyDirlist)
		return
	}

	dt.UpdateScroller(ctx.Height(), n)
	dt.EnsureScroll(dt.countVisible(dt.list[:dt.listIdx]))

	needScrollbar := true
	percentVisible := float64(ctx.Height()) / float64(n)
	if percentVisible >= 1.0 {
		needScrollbar = false
	}

	textWidth := ctx.Width()
	if needScrollbar {
		textWidth -= 1
	}
	if textWidth < 0 {
		return
	}

	treeCtx := ctx.Subcontext(0, 0, textWidth, ctx.Height())

	data := state.NewDataSetter()
	data.SetAccount(dt.acctConf)

	n = 0
	for i, node := range dt.list {
		if n > treeCtx.Height() {
			break
		}
		rowNr := dt.countVisible(dt.list[:i])
		if rowNr < dt.Scroll() || !isVisible(node) {
			continue
		}

		path := dt.getDirectory(node)
		dir := dt.Directory(path)
		treeDir := &models.Directory{
			Name: dt.displayText(node),
		}
		if dir != nil {
			treeDir.Role = dir.Role
		}
		data.SetFolder(treeDir)
		data.SetRUE([]string{path}, dt.GetRUECount)

		left, right, style := dt.renderDir(
			path, uiConfig, data.Data(),
			i == dt.listIdx, treeCtx.Width(),
		)

		treeCtx.Printf(0, n, style, "%s %s", left, right)
		n++
	}

	if dt.NeedScrollbar() {
		scrollBarCtx := ctx.Subcontext(ctx.Width()-1, 0, 1, ctx.Height())
		dt.drawScrollbar(scrollBarCtx)
	}
}

func (dt *DirectoryTree) MouseEvent(localX int, localY int, event vaxis.Event) {
	if event, ok := event.(vaxis.Mouse); ok {
		switch event.Button {
		case vaxis.MouseLeftButton:
			clickedDir, ok := dt.Clicked(localX, localY)
			if ok {
				dt.Select(clickedDir)
			}
		case vaxis.MouseWheelDown:
			dt.NextPrev(1)
		case vaxis.MouseWheelUp:
			dt.NextPrev(-1)
		}
	}
}

func (dt *DirectoryTree) Clicked(x int, y int) (string, bool) {
	if len(dt.list) == 0 || dt.countVisible(dt.list) < y+dt.Scroll() {
		return "", false
	}
	visible := 0
	for _, node := range dt.list {
		if isVisible(node) {
			visible++
		}
		if visible == y+dt.Scroll()+1 {
			if path := dt.getDirectory(node); path != "" {
				return path, true
			}
			if node.Hidden == 0 {
				node.Hidden = 1
			} else {
				node.Hidden = 0
			}
			dt.Invalidate()
			return "", false
		}
	}
	return "", false
}

func (dt *DirectoryTree) SelectedMsgStore() (*lib.MessageStore, bool) {
	if dt.virtual {
		return nil, false
	}

	selected := models.UID(dt.selected)
	if _, node := dt.getTreeNode(selected); node == nil {
		dt.buildTree()
		selIdx, node := dt.getTreeNode(selected)
		if node != nil {
			makeVisible(node)
			dt.listIdx = selIdx
		}
	}
	return dt.DirectoryList.SelectedMsgStore()
}

func (dt *DirectoryTree) reindex(name string) {
	selIdx, node := dt.getTreeNode(models.UID(name))
	if node != nil {
		makeVisible(node)
		dt.listIdx = selIdx
	}
}

func (dt *DirectoryTree) Select(name string) {
	if name == "" {
		return
	}
	dt.Open(name, "", dt.UiConfig(name).DirListDelay, nil, false)
}

func (dt *DirectoryTree) Open(name string, query string, delay time.Duration, cb func(types.WorkerMessage), force bool) {
	if name == "" {
		return
	}
	again := false
	uid := models.UID(name)
	if _, node := dt.getTreeNode(uid); node == nil {
		again = true
	} else {
		dt.reindex(name)
	}
	dt.DirectoryList.Open(name, query, delay, func(msg types.WorkerMessage) {
		if cb != nil {
			cb(msg)
		}
		if _, ok := msg.(*types.Done); ok && again {
			if findString(dt.dirs, name) < 0 {
				dt.dirs = append(dt.dirs, name)
			}
			dt.buildTree()
			dt.reindex(name)
		}
	}, force)
}

func (dt *DirectoryTree) NextPrev(delta int) {
	newIdx := dt.listIdx
	ndirs := len(dt.list)
	if newIdx == ndirs {
		return
	}

	if ndirs == 0 {
		return
	}

	step := 1
	if delta < 0 {
		step = -1
		delta *= -1
	}

	for i := 0; i < delta; {
		newIdx += step
		if newIdx < 0 {
			newIdx = ndirs - 1
		} else if newIdx >= ndirs {
			newIdx = 0
		}
		if isVisible(dt.list[newIdx]) {
			i++
		}
	}

	dt.selectIndex(newIdx)
}

func (dt *DirectoryTree) selectIndex(i int) {
	dt.listIdx = i
	node := dt.list[dt.listIdx]
	if node.Dummy {
		dt.virtual = true
		dt.NewContext()
		dt.virtualCb()
	} else {
		dt.virtual = false
		dt.Select(dt.getDirectory(node))
	}
}

func (dt *DirectoryTree) CollapseFolder() {
	if dt.listIdx >= 0 && dt.listIdx < len(dt.list) {
		if node := dt.list[dt.listIdx]; node != nil {
			if node.Parent != nil && (node.Hidden != 0 || node.FirstChild == nil) {
				node.Parent.Hidden = 1
				// highlight parent node and select it
				for i, t := range dt.list {
					if t == node.Parent {
						dt.selectIndex(i)
					}
				}
			} else {
				node.Hidden = 1
			}
			dt.Invalidate()
		}
	}
}

func (dt *DirectoryTree) ExpandFolder() {
	if dt.listIdx >= 0 && dt.listIdx < len(dt.list) {
		dt.list[dt.listIdx].Hidden = 0
		dt.Invalidate()
	}
}

func (dt *DirectoryTree) countVisible(list []*types.Thread) (n int) {
	for _, node := range list {
		if isVisible(node) {
			n++
		}
	}
	return
}

func (dt *DirectoryTree) nodeElems(node *types.Thread) []string {
	dir := string(node.Uid)
	sep := dt.DirectoryList.worker.PathSeparator()
	return strings.Split(dir, sep)
}

func (dt *DirectoryTree) nodeName(node *types.Thread) string {
	if elems := dt.nodeElems(node); len(elems) > 0 {
		return elems[len(elems)-1]
	}
	return ""
}

func (dt *DirectoryTree) displayText(node *types.Thread) string {
	return fmt.Sprintf("%s%s%s",
		threadPrefix(node, false, false),
		getFlag(node), dt.nodeName(node))
}

func (dt *DirectoryTree) getDirectory(node *types.Thread) string {
	return string(node.Uid)
}

func (dt *DirectoryTree) getTreeNode(uid models.UID) (int, *types.Thread) {
	for i, node := range dt.list {
		if node.Uid == uid {
			return i, node
		}
	}
	return -1, nil
}

func (dt *DirectoryTree) hiddenDirectories() map[string]bool {
	hidden := make(map[string]bool, 0)
	for _, node := range dt.list {
		if node.Hidden != 0 && node.FirstChild != nil {
			elems := dt.nodeElems(node)
			if levels := countLevels(node); levels < len(elems) {
				if node.FirstChild != nil && (levels+1) < len(elems) {
					levels += 1
				}
				if dirStr := strings.Join(elems[:levels], dt.DirectoryList.worker.PathSeparator()); dirStr != "" {
					hidden[dirStr] = true
				}
			}
		}
	}
	return hidden
}

func (dt *DirectoryTree) setHiddenDirectories(hiddenDirs map[string]bool) {
	log.Tracef("setHiddenDirectories: %#v", hiddenDirs)
	for _, node := range dt.list {
		elems := dt.nodeElems(node)
		if levels := countLevels(node); levels < len(elems) {
			if node.FirstChild != nil && (levels+1) < len(elems) {
				levels += 1
			}
			strDir := strings.Join(elems[:levels], dt.DirectoryList.worker.PathSeparator())
			if hidden, ok := hiddenDirs[strDir]; hidden && ok {
				node.Hidden = 1
				log.Tracef("setHiddenDirectories: %q -> %#v", strDir, node)
			}
		}
	}
}

func (dt *DirectoryTree) buildTree() {
	if len(dt.list) != 0 {
		hiddenDirs := dt.hiddenDirectories()
		defer dt.setHiddenDirectories(hiddenDirs)
	}

	dirs := make([]string, len(dt.dirs))
	copy(dirs, dt.dirs)
	root := &types.Thread{}
	dt.buildTreeNode(root, dirs, 1)

	var threads []*types.Thread
	for iter := root.FirstChild; iter != nil; iter = iter.NextSibling {
		iter.Parent = nil
		threads = append(threads, iter)
	}

	// folders-sort
	if dt.DirectoryList.acctConf.EnableFoldersSort {
		sort.Slice(threads, func(i, j int) bool {
			foldersSort := dt.DirectoryList.acctConf.FoldersSort
			iInFoldersSort := findString(foldersSort, dt.getDirectory(threads[i]))
			jInFoldersSort := findString(foldersSort, dt.getDirectory(threads[j]))
			if iInFoldersSort >= 0 && jInFoldersSort >= 0 {
				return iInFoldersSort < jInFoldersSort
			}
			if iInFoldersSort >= 0 {
				return true
			}
			if jInFoldersSort >= 0 {
				return false
			}
			return dt.getDirectory(threads[i]) < dt.getDirectory(threads[j])
		})
	}

	dt.list = make([]*types.Thread, 0)
	for _, node := range threads {
		err := node.Walk(func(t *types.Thread, lvl int, err error) error {
			dt.list = append(dt.list, t)
			return nil
		})
		if err != nil {
			log.Warnf("failed to walk tree: %v", err)
		}
	}
}

func (dt *DirectoryTree) buildTreeNode(node *types.Thread, dirs []string, depth int) {
	dirmap := make(map[string][]string)
	for _, dir := range dirs {
		base, dir, cut := strings.Cut(
			dir, dt.DirectoryList.worker.PathSeparator())
		if _, found := dirmap[base]; found {
			if cut {
				dirmap[base] = append(dirmap[base], dir)
			}
		} else if cut {
			dirmap[base] = append(dirmap[base], dir)
		} else {
			dirmap[base] = []string{}
		}
	}
	bases := make([]string, 0, len(dirmap))
	for base, dirs := range dirmap {
		bases = append(bases, base)
		sort.Strings(dirs)
	}
	sort.Strings(bases)

	basePath := dt.getDirectory(node)
	if depth > dt.UiConfig(basePath).DirListCollapse {
		node.Hidden = 1
	} else {
		node.Hidden = 0
	}

	for _, base := range bases {
		path := dt.childPath(basePath, base)
		nextNode := &types.Thread{Uid: models.UID(path)}

		nextNode.Dummy = findString(dt.dirs, path) == -1

		node.AddChild(nextNode)
		dt.buildTreeNode(nextNode, dirmap[base], depth+1)
	}
}

func (dt *DirectoryTree) childPath(base, relpath string) string {
	if base == "" {
		return relpath
	}
	return base + dt.DirectoryList.worker.PathSeparator() + relpath
}

func makeVisible(node *types.Thread) {
	if node == nil {
		return
	}
	for iter := node.Parent; iter != nil; iter = iter.Parent {
		iter.Hidden = 0
	}
}

func isVisible(node *types.Thread) bool {
	for iter := node.Parent; iter != nil; iter = iter.Parent {
		if iter.Hidden != 0 {
			return false
		}
	}
	return true
}

func countLevels(node *types.Thread) (level int) {
	for iter := node.Parent; iter != nil; iter = iter.Parent {
		level++
	}
	return
}

func getFlag(node *types.Thread) string {
	if node == nil || node.FirstChild == nil {
		return ""
	}
	if node.Hidden != 0 {
		return "+"
	}
	return ""
}
