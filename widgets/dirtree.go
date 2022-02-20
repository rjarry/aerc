package widgets

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/gdamore/tcell/v2"
)

type DirectoryTree struct {
	*DirectoryList

	listIdx int
	list    []*types.Thread

	pathSeparator string
	treeDirs      []string
}

func NewDirectoryTree(dirlist *DirectoryList, pathSeparator string) DirectoryLister {
	dt := &DirectoryTree{
		DirectoryList: dirlist,
		listIdx:       -1,
		list:          make([]*types.Thread, 0),
		pathSeparator: pathSeparator,
	}
	return dt
}

func (dt *DirectoryTree) UpdateList(done func([]string)) {
	dt.DirectoryList.UpdateList(func(dirs []string) {
		if done != nil {
			done(dirs)
		}
		dt.buildTree()
		dt.listIdx = findString(dt.dirs, dt.selecting)
		dt.Select(dt.selecting)
		dt.scroll = 0
	})
}

func (dt *DirectoryTree) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		dt.UiConfig().GetStyle(config.STYLE_DIRLIST_DEFAULT))

	if dt.DirectoryList.spinner.IsRunning() {
		dt.DirectoryList.spinner.Draw(ctx)
		return
	}

	n := dt.countVisible(dt.list)
	if n == 0 {
		style := dt.UiConfig().GetStyle(config.STYLE_DIRLIST_DEFAULT)
		ctx.Printf(0, 0, style, dt.UiConfig().EmptyDirlist)
		return
	}

	dt.ensureScroll(ctx.Height())

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
		textWidth = 0
	}

	rowNr := 0
	for i, node := range dt.list {
		if i < dt.scroll || !isVisible(node) {
			continue
		}
		row := rowNr - dt.scroll
		if row >= ctx.Height() {
			break
		}

		name := dt.displayText(node)
		rowNr++

		style := dt.UiConfig().GetStyle(config.STYLE_DIRLIST_DEFAULT)
		if i == dt.listIdx {
			style = dt.UiConfig().GetStyleSelected(config.STYLE_DIRLIST_DEFAULT)
		}
		ctx.Fill(0, row, textWidth, 1, ' ', style)

		dirString := dt.getDirString(name, textWidth, func() string {
			if path := dt.getDirectory(node); path != "" {
				return dt.getRUEString(path)
			}
			return ""
		})

		ctx.Printf(0, row, style, dirString)
	}

	if needScrollbar {
		scrollBarCtx := ctx.Subcontext(ctx.Width()-1, 0, 1, ctx.Height())
		dt.drawScrollbar(scrollBarCtx, percentVisible)
	}
}

func (dt *DirectoryTree) ensureScroll(h int) {
	selectingIdx := dt.countVisible(dt.list[:dt.listIdx])
	if selectingIdx < 0 {
		// dir not found, meaning we are currently adding / removing a dir.
		// we can simply ignore this until we get redrawn with the new
		// dirlist.dir content
		return
	}

	maxScroll := dt.countVisible(dt.list) - h
	if maxScroll < 0 {
		maxScroll = 0
	}

	if selectingIdx >= dt.scroll && selectingIdx < dt.scroll+h {
		if dt.scroll > maxScroll {
			dt.scroll = maxScroll
		}
		return
	}

	if selectingIdx >= dt.scroll+h {
		dt.scroll = selectingIdx - h + 1
	} else if selectingIdx < dt.scroll {
		dt.scroll = selectingIdx
	}

	if dt.scroll > maxScroll {
		dt.scroll = maxScroll
	}
}

func (dt *DirectoryTree) MouseEvent(localX int, localY int, event tcell.Event) {
	switch event := event.(type) {
	case *tcell.EventMouse:
		switch event.Buttons() {
		case tcell.Button1:
			clickedDir, ok := dt.Clicked(localX, localY)
			if ok {
				dt.Select(clickedDir)
			}
		case tcell.WheelDown:
			dt.Next()
		case tcell.WheelUp:
			dt.Prev()
		}
	}
}

func (dt *DirectoryTree) Clicked(x int, y int) (string, bool) {
	if dt.list == nil || len(dt.list) == 0 || dt.countVisible(dt.list) < y {
		return "", false
	}
	for i, node := range dt.list {
		if dt.countVisible(dt.list[:i]) == y {
			if path := dt.getDirectory(node); path != "" {
				return path, true
			}
		}
	}
	return "", false
}

func (dt *DirectoryTree) Select(name string) {
	idx := findString(dt.treeDirs, name)
	if idx >= 0 {
		selIdx, node := dt.getTreeNode(uint32(idx))
		if node != nil {
			makeVisible(node)
			dt.listIdx = selIdx
		}
	}

	if name == "" {
		return
	}

	dt.DirectoryList.Select(name)
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
		newIdx = newIdx + step
		if newIdx < 0 {
			newIdx = ndirs - 1
		} else if newIdx >= ndirs {
			newIdx = 0
		}
		if isVisible(dt.list[newIdx]) {
			i++
		}
	}

	dt.listIdx = newIdx
	if path := dt.getDirectory(dt.list[dt.listIdx]); path != "" {
		dt.Select(path)
	}
}

func (dt *DirectoryTree) CollapseFolder() {
	if dt.listIdx >= 0 && dt.listIdx < len(dt.list) {
		if node := dt.list[dt.listIdx]; node != nil {
			if node.Parent != nil && (node.Hidden || node.FirstChild == nil) {
				node.Parent.Hidden = true
				// highlight parent node and select it
				for i, t := range dt.list {
					if t == node.Parent {
						dt.listIdx = i
						if path := dt.getDirectory(dt.list[dt.listIdx]); path != "" {
							dt.Select(path)
						}
					}
				}
			} else {
				node.Hidden = true
			}
			dt.Invalidate()
		}
	}
}

func (dt *DirectoryTree) ExpandFolder() {
	if dt.listIdx >= 0 && dt.listIdx < len(dt.list) {
		dt.list[dt.listIdx].Hidden = false
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

func (dt *DirectoryTree) displayText(node *types.Thread) string {
	elems := strings.Split(dt.treeDirs[getAnyUid(node)], dt.pathSeparator)
	return fmt.Sprintf("%s%s%s", threadPrefix(node), getFlag(node), elems[countLevels(node)])
}

func (dt *DirectoryTree) getDirectory(node *types.Thread) string {
	if uid := node.Uid; uid >= 0 && int(uid) < len(dt.treeDirs) {
		return dt.treeDirs[uid]
	}
	return ""
}

func (dt *DirectoryTree) getTreeNode(uid uint32) (int, *types.Thread) {
	var found *types.Thread
	var idx int
	for i, node := range dt.list {
		if node.Uid == uid {
			found = node
			idx = i
		}
	}
	return idx, found
}

func (dt *DirectoryTree) hiddenDirectories() map[string]bool {
	hidden := make(map[string]bool, 0)
	for _, node := range dt.list {
		if node.Hidden && node.FirstChild != nil {
			elems := strings.Split(dt.treeDirs[getAnyUid(node)], dt.pathSeparator)
			if levels := countLevels(node); levels < len(elems) {
				if node.FirstChild != nil && (levels+1) < len(elems) {
					levels += 1
				}
				if dirStr := strings.Join(elems[:levels], dt.pathSeparator); dirStr != "" {
					hidden[dirStr] = true
				}
			}
		}
	}
	return hidden
}

func (dt *DirectoryTree) setHiddenDirectories(hiddenDirs map[string]bool) {
	for _, node := range dt.list {
		elems := strings.Split(dt.treeDirs[getAnyUid(node)], dt.pathSeparator)
		if levels := countLevels(node); levels < len(elems) {
			if node.FirstChild != nil && (levels+1) < len(elems) {
				levels += 1
			}
			strDir := strings.Join(elems[:levels], dt.pathSeparator)
			if hidden, ok := hiddenDirs[strDir]; hidden && ok {
				node.Hidden = true
			}
		}
	}
}

func (dt *DirectoryTree) buildTree() {
	if len(dt.list) != 0 {
		hiddenDirs := dt.hiddenDirectories()
		defer func() {
			dt.setHiddenDirectories(hiddenDirs)
		}()
	}

	sTree := make([][]string, 0)
	for i, dir := range dt.dirs {
		elems := strings.Split(dir, dt.pathSeparator)
		if len(elems) == 0 {
			continue
		}
		elems = append(elems, fmt.Sprintf("%d", i))
		sTree = append(sTree, elems)
	}

	dt.treeDirs = make([]string, len(dt.dirs))
	copy(dt.treeDirs, dt.dirs)

	root := &types.Thread{Uid: 0}
	buildTree(root, sTree, 0xFFFFFF)

	threads := make([]*types.Thread, 0)

	for iter := root.FirstChild; iter != nil; iter = iter.NextSibling {
		iter.Parent = nil
		threads = append(threads, iter)
	}

	// folders-sort
	if dt.DirectoryList.acctConf.EnableFoldersSort {
		toStr := func(t *types.Thread) string {
			if elems := strings.Split(dt.treeDirs[getAnyUid(t)], dt.pathSeparator); len(elems) > 0 {
				return elems[0]
			}
			return ""
		}
		sort.Slice(threads, func(i, j int) bool {
			foldersSort := dt.DirectoryList.acctConf.FoldersSort
			iInFoldersSort := findString(foldersSort, toStr(threads[i]))
			jInFoldersSort := findString(foldersSort, toStr(threads[j]))
			if iInFoldersSort >= 0 && jInFoldersSort >= 0 {
				return iInFoldersSort < jInFoldersSort
			}
			if iInFoldersSort >= 0 {
				return true
			}
			if jInFoldersSort >= 0 {
				return false
			}
			return toStr(threads[i]) < toStr(threads[j])
		})
	}

	dt.list = make([]*types.Thread, 0)
	for _, node := range threads {
		node.Walk(func(t *types.Thread, lvl int, err error) error {
			dt.list = append(dt.list, t)
			return nil
		})
	}
}

func buildTree(node *types.Thread, stree [][]string, defaultUid uint32) {
	m := make(map[string][][]string)
	for _, branch := range stree {
		if len(branch) > 1 {
			var next [][]string
			if _, ok := m[branch[0]]; !ok {
				next = make([][]string, 0)
			}
			next = append(m[branch[0]], branch[1:])
			m[branch[0]] = next
		}
	}
	keys := make([]string, 0)
	for key, _ := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		next, _ := m[key]
		var uid uint32 = defaultUid
		for _, testStr := range next {
			if len(testStr) == 1 {
				if uidI, err := strconv.Atoi(next[0][0]); err == nil {
					uid = uint32(uidI)
				}
			}
		}
		nextNode := &types.Thread{Uid: uid}
		node.AddChild(nextNode)
		buildTree(nextNode, next, defaultUid)
	}
}

func makeVisible(node *types.Thread) {
	if node == nil {
		return
	}
	for iter := node.Parent; iter != nil; iter = iter.Parent {
		iter.Hidden = false
	}
}

func isVisible(node *types.Thread) bool {
	isVisible := true
	for iter := node.Parent; iter != nil; iter = iter.Parent {
		if iter.Hidden {
			isVisible = false
			break
		}
	}
	return isVisible
}

func getAnyUid(node *types.Thread) (uid uint32) {
	node.Walk(func(t *types.Thread, l int, err error) error {
		if t.FirstChild == nil {
			uid = t.Uid
		}
		return nil
	})
	return
}

func countLevels(node *types.Thread) (level int) {
	for iter := node.Parent; iter != nil; iter = iter.Parent {
		level++
	}
	return
}

func getFlag(node *types.Thread) (flag string) {
	if node != nil && node.FirstChild != nil {
		if node.Hidden {
			flag = "─"
		} else {
			flag = "┌"
		}
	}
	return
}
