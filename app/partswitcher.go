package app

import (
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type PartSwitcher struct {
	parts          []*PartViewer
	selected       int
	alwaysShowMime bool

	height int
	mv     *MessageViewer
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

func (ps *PartSwitcher) Event(event tcell.Event) bool {
	return ps.parts[ps.selected].Event(event)
}

func (ps *PartSwitcher) Draw(ctx *ui.Context) {
	height := len(ps.parts)
	if height == 1 && !config.Viewer.AlwaysShowMime {
		ps.parts[ps.selected].Draw(ctx)
		return
	}

	var styleSwitcher, styleFile, styleMime tcell.Style

	// TODO: cap height and add scrolling for messages with many parts
	ps.height = ctx.Height()
	y := ctx.Height() - height
	for i, part := range ps.parts {
		if ps.selected == i {
			styleSwitcher = ps.mv.uiConfig.GetStyleSelected(config.STYLE_PART_SWITCHER)
			styleFile = ps.mv.uiConfig.GetStyleSelected(config.STYLE_PART_FILENAME)
			styleMime = ps.mv.uiConfig.GetStyleSelected(config.STYLE_PART_MIMETYPE)
		} else {
			styleSwitcher = ps.mv.uiConfig.GetStyle(config.STYLE_PART_SWITCHER)
			styleFile = ps.mv.uiConfig.GetStyle(config.STYLE_PART_FILENAME)
			styleMime = ps.mv.uiConfig.GetStyle(config.STYLE_PART_MIMETYPE)
		}
		ctx.Fill(0, y+i, ctx.Width(), 1, ' ', styleSwitcher)
		left := len(part.index) * 2
		if part.part.FileName() != "" {
			name := runewidth.Truncate(part.part.FileName(),
				ctx.Width()-left-1, "…")
			left += ctx.Printf(left, y+i, styleFile, "%s ", name)
		}
		t := "(" + part.part.FullMIMEType() + ")"
		t = runewidth.Truncate(t, ctx.Width()-left, "…")
		ctx.Printf(left, y+i, styleMime, "%s", t)
	}
	ps.parts[ps.selected].Draw(ctx.Subcontext(
		0, 0, ctx.Width(), ctx.Height()-height))
}

func (ps *PartSwitcher) MouseEvent(localX int, localY int, event tcell.Event) {
	if event, ok := event.(*tcell.EventMouse); ok {
		switch event.Buttons() {
		case tcell.Button1:
			height := len(ps.parts)
			y := ps.height - height
			if localY < y && ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
			}
			for i := range ps.parts {
				if localY != y+i {
					continue
				}
				if ps.parts[i].part.MIMEType == "multipart" {
					continue
				}
				if ps.parts[ps.selected].term != nil {
					ps.parts[ps.selected].term.Focus(false)
				}
				ps.selected = i
				ps.Invalidate()
				if ps.parts[ps.selected].term != nil {
					ps.parts[ps.selected].term.Focus(true)
				}
			}
		case tcell.WheelDown:
			height := len(ps.parts)
			y := ps.height - height
			if localY < y && ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
			}
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(false)
			}
			ps.mv.NextPart()
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(true)
			}
		case tcell.WheelUp:
			height := len(ps.parts)
			y := ps.height - height
			if localY < y && ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.MouseEvent(localX, localY, event)
			}
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(false)
			}
			ps.mv.PreviousPart()
			if ps.parts[ps.selected].term != nil {
				ps.parts[ps.selected].term.Focus(true)
			}
		}
	}
}

func (ps *PartSwitcher) Cleanup() {
	for _, partViewer := range ps.parts {
		partViewer.Cleanup()
	}
}
