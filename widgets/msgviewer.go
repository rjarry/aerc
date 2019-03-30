package widgets

import (
	"bytes"
	"io"
	"os/exec"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type MessageViewer struct {
	grid *ui.Grid
	term *Terminal
}

var testMsg = `Makes the following changes to the Event type:

* make 'user' and 'ticket' nullable since some events require it
* add 'by_user' and 'from_ticket' to enable mentions
* remove 'assinged_user' which is no longer used

Ticket: https://todo.sr.ht/~sircmpwn/todo.sr.ht/156
---
 tests/test_comments.py                        |  23 ++-
 .../versions/75ff2f7624fd_new_event_fields.py | 142 ++++++++++++++++++
 todosrht/templates/events.html                |  18 ++-
 todosrht/templates/ticket.html                |  31 +++-
 todosrht/tickets.py                           |  14 +-
 todosrht/types/event.py                       |  16 +-
 6 files changed, 207 insertions(+), 37 deletions(-)
 create mode 100644 todosrht/alembic/versions/75ff2f7624fd_new_event_fields.py

diff --git a/tests/test_comments.py b/tests/test_comments.py
index 4b3161d..b85d751 100644
--- a/tests/test_comments.py
+++ b/tests/test_comments.py
@@ -253,20 +253,25 @@ def test_notifications_and_events(mailbox):
     # Check correct events are generated
     comment_events = {e for e in ticket.events
         if e.event_type == EventType.comment}
-    user_events = {e for e in ticket.events
+    u1_events = {e for e in u1.events
+        if e.event_type == EventType.user_mentioned}
+    u2_events = {e for e in u2.events
         if e.event_type == EventType.user_mentioned}

     assert len(comment_events) == 1
-    assert len(user_events) == 2
+    assert len(u1_events) == 1
+    assert len(u2_events) == 1

-    u1_mention = next(e for e in user_events if e.user == u1)
-    u2_mention = next(e for e in user_events if e.user == u2)
+    u1_mention = u1_events.pop()
+    u2_mention = u2_events.pop()

     assert u1_mention.comment == comment
-    assert u1_mention.ticket == ticket
+    assert u1_mention.from_ticket == ticket
+    assert u1_mention.by_user == commenter

     assert u2_mention.comment == comment
-    assert u2_mention.ticket == ticket
+    assert u2_mention.from_ticket == ticket
+    assert u2_mention.by_user == commenter

     assert len(t1.events) == 1
     assert len(t2.events) == 1
@@ -276,10 +281,12 @@ def test_notifications_and_events(mailbox):
     t2_mention = t2.events[0]

     assert t1_mention.comment == comment
-    assert t1_mention.user == commenter
+    assert t1_mention.from_ticket == ticket
+    assert t1_mention.by_user == commenter

     assert t2_mention.comment == comment
-    assert t2_mention.user == commenter
+    assert t2_mention.from_ticket == ticket
+    assert t2_mention.by_user == commenter

 def test_ticket_mention_pattern():
     def match(text):
diff --git a/todosrht/alembic/versions/75ff2f7624fd_new_event_fields.py
b/todosrht/alembic/versions/75ff2f7624fd_new_event_fields.py
new file mode 100644
index 0000000..1c55bfe
--- /dev/null
+++ b/todosrht/alembic/versions/75ff2f7624fd_new_event_fields.py
@@ -0,0 +1,142 @@
+"""Add new event fields and migrate data.
+
+Also makes Event.ticket_id and Event.user_id nullable since some these fields
+can be empty for mention events.
+
+Revision ID: 75ff2f7624fd
+Revises: c7146cb70d6b
+Create Date: 2019-03-28 16:26:18.714300
+
+"""
+
+# revision identifiers, used by Alembic.
+revision = "75ff2f7624fd"
+down_revision = "c7146cb70d6b"
`

func NewMessageViewer() *MessageViewer {
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 3},
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})

	headers := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 1},
		{ui.SIZE_EXACT, 1},
		{ui.SIZE_EXACT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_WEIGHT, 1},
	})
	headers.AddChild(
		&HeaderView{
			Name:  "From",
			Value: "Ivan Habunek <ivan@habunek.com>",
		}).At(0, 0)
	headers.AddChild(
		&HeaderView{
			Name:  "To",
			Value: "~sircmpwn/sr.ht-dev@lists.sr.ht",
		}).At(0, 1)
	headers.AddChild(
		&HeaderView{
			Name: "Subject",
			Value: "[PATCH todo.sr.ht v2 1/3 Alter Event fields " +
				"and migrate data]",
		}).At(1, 0).Span(1, 2)
	headers.AddChild(ui.NewFill(' ')).At(2, 0).Span(1, 2)

	cmd := exec.Command("sh", "-c", "./contrib/hldiff.py | less -R")
	pipe, _ := cmd.StdinPipe()
	term, _ := NewTerminal(cmd)
	term.OnStart = func() {
		go func() {
			reader := bytes.NewBufferString(testMsg)
			io.Copy(pipe, reader)
			pipe.Close()
		}()
	}
	term.Focus(true)

	grid.AddChild(headers).At(0, 0)
	grid.AddChild(term).At(1, 0)
	return &MessageViewer{grid, term}
}

func (mv *MessageViewer) Draw(ctx *ui.Context) {
	mv.grid.Draw(ctx)
}

func (mv *MessageViewer) Invalidate() {
	mv.grid.Invalidate()
}

func (mv *MessageViewer) OnInvalidate(fn func(d ui.Drawable)) {
	mv.grid.OnInvalidate(func(_ ui.Drawable) {
		fn(mv)
	})
}

func (mv *MessageViewer) Event(event tcell.Event) bool {
	return mv.term.Event(event)
}

func (mv *MessageViewer) Focus(focus bool) {
	mv.term.Focus(focus)
}

type HeaderView struct {
	onInvalidate func(d ui.Drawable)

	Name  string
	Value string
}

func (hv *HeaderView) Draw(ctx *ui.Context) {
	size := runewidth.StringWidth(" " + hv.Name + " ")
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	style := tcell.StyleDefault.Reverse(true)
	ctx.Printf(0, 0, style, " "+hv.Name+" ")
	style = tcell.StyleDefault
	ctx.Printf(size, 0, style, " "+hv.Value)
}

func (hv *HeaderView) Invalidate() {
	if hv.onInvalidate != nil {
		hv.onInvalidate(hv)
	}
}

func (hv *HeaderView) OnInvalidate(fn func(d ui.Drawable)) {
	hv.onInvalidate = fn
}
