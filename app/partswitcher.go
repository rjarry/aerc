package app

import (
	"math"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type PartSwitcher struct {
	Scrollable
	parts    []*PartViewer
	selected int

	height int
	offset int

	uiConfig *config.UIConfig
}

func (ps *PartSwitcher) PreviousPart() {
	for {
		ps.selected--
		if ps.selected < 0 {
			ps.selected = len(ps.parts) - 1
		}
		if ps.parts[ps.selected].part.MIMEType != "multipart" {
			break
		}
	}
}

func (ps *PartSwitcher) NextPart() {
	for {
		ps.selected++
		if ps.selected >= len(ps.parts) {
			ps.selected = 0
		}
		if ps.parts[ps.selected].part.MIMEType != "multipart" {
			break
		}
	}
}

func (ps *PartSwitcher) SelectedPart() *PartViewer {
	return ps.parts[ps.selected]
}

func (ps *PartSwitcher) AttachmentParts(all bool) []*PartInfo {
	var attachments []*PartInfo
	for _, p := range ps.parts {
		if p.part.Disposition == "attachment" || (all && p.part.FileName() != "") {
			pi := &PartInfo{
				Index: p.index,
				Msg:   p.msg.MessageInfo(),
				Part:  p.part,
			}
			attachments = append(attachments, pi)
		}
	}
	return attachments
}

func (ps *PartSwitcher) Invalidate() {
	ui.Invalidate()
}

func (ps *PartSwitcher) Focus(focus bool) {
	if ps.parts[ps.selected].term != nil {
		ps.parts[ps.selected].term.Focus(focus)
	}
}

func (ps *PartSwitcher) Show(visible bool) {
	if ps.parts[ps.selected].term != nil {
		ps.parts[ps.selected].term.Show(visible)
	}
}

func (ps *PartSwitcher) Event(event vaxis.Event) bool {
	return ps.parts[ps.selected].Event(event)
}

func (ps *PartSwitcher) Draw(ctx *ui.Context) {
	uiConfig := ps.uiConfig
	n := len(ps.parts)
	if n == 1 && !config.Viewer.AlwaysShowMime {
		ps.parts[ps.selected].Draw(ctx)
		return
	}

	ps.height = config.Viewer.MaxMimeHeight
	if ps.height <= 0 || n < ps.height {
		ps.height = n
	}
	if ps.height > ctx.Height()/2 {
		ps.height = ctx.Height() / 2
	}

	ps.UpdateScroller(ps.height, n)
	ps.EnsureScroll(ps.selected)

	var styleSwitcher, styleFile, styleMime vaxis.Style

	scrollbarWidth := 0
	if ps.NeedScrollbar() {
		scrollbarWidth = 1
	}

	ps.offset = ctx.Height() - ps.height
	y := ps.offset
	row := ps.offset
	ctx.Fill(0, y, ctx.Width(), ps.height, ' ', uiConfig.GetStyle(config.STYLE_PART_SWITCHER))
	for i := ps.Scroll(); i < n; i++ {
		part := ps.parts[i]
		if ps.selected == i {
			styleSwitcher = uiConfig.GetStyleSelected(config.STYLE_PART_SWITCHER)
			styleFile = uiConfig.GetStyleSelected(config.STYLE_PART_FILENAME)
			styleMime = uiConfig.GetStyleSelected(config.STYLE_PART_MIMETYPE)
		} else {
			styleSwitcher = uiConfig.GetStyle(config.STYLE_PART_SWITCHER)
			styleFile = uiConfig.GetStyle(config.STYLE_PART_FILENAME)
			styleMime = uiConfig.GetStyle(config.STYLE_PART_MIMETYPE)
		}
		ctx.Fill(0, row, ctx.Width(), 1, ' ', styleSwitcher)
		left := len(part.index) * 2
		if part.part.FileName() != "" {
			name := runewidth.Truncate(part.part.FileName(),
				ctx.Width()-left-1, "…")
			left += ctx.Printf(left, row, styleFile, "%s ", name)
		}
		t := "(" + part.part.FullMIMEType() + ")"
		t = runewidth.Truncate(t, ctx.Width()-left-scrollbarWidth, "…")
		ctx.Printf(left, row, styleMime, "%s", t)
		row++

		if (i - ps.Scroll()) >= ps.height {
			break
		}
	}
	if ps.NeedScrollbar() {
		ps.drawScrollbar(ctx.Subcontext(ctx.Width()-1, y, 1, ps.height))
	}
	ps.parts[ps.selected].Draw(ctx.Subcontext(
		0, 0, ctx.Width(), ctx.Height()-ps.height))
}

func (ps *PartSwitcher) drawScrollbar(ctx *ui.Context) {
	uiConfig := ps.uiConfig
	gutterStyle := uiConfig.GetStyle(config.STYLE_MSGLIST_GUTTER)
	pillStyle := uiConfig.GetStyle(config.STYLE_MSGLIST_PILL)

	// gutter
	ctx.Fill(0, 0, 1, ctx.Height(), ' ', gutterStyle)

	// pill
	pillSize := int(math.Ceil(float64(ctx.Height()) * ps.PercentVisible()))
	pillOffset := int(math.Floor(float64(ctx.Height()) * ps.PercentScrolled()))
	ctx.Fill(0, pillOffset, 1, pillSize, ' ', pillStyle)
}

func (ps *PartSwitcher) MouseEvent(localX int, localY int, event vaxis.Event) {
	if localY < ps.offset && ps.parts[ps.selected].term != nil {
		ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
		return
	}

	e, ok := event.(*tcell.EventMouse)
	if !ok {
		return
	}

	if ps.parts[ps.selected].term != nil {
		ps.parts[ps.selected].term.Focus(false)
	}

	switch e.Buttons() {
	case tcell.Button1:
		i := localY - ps.offset + ps.Scroll()
		if i < 0 || i >= len(ps.parts) {
			break
		}
		if ps.parts[i].part.MIMEType == "multipart" {
			break
		}
		ps.selected = i
		ps.Invalidate()
	case tcell.WheelDown:
		ps.NextPart()
		ps.Invalidate()
	case tcell.WheelUp:
		ps.PreviousPart()
		ps.Invalidate()
	}

	if ps.parts[ps.selected].term != nil {
		ps.parts[ps.selected].term.Focus(true)
	}
}

func (ps *PartSwitcher) Cleanup() {
	for _, partViewer := range ps.parts {
		partViewer.Cleanup()
	}
}
