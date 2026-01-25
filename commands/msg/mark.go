package msg

import (
	"fmt"
	"iter"
	"slices"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type Mark struct {
	All             bool   `opt:"-a" aliases:"mark,unmark" desc:"Mark all messages in current folder."`
	Toggle          bool   `opt:"-t" aliases:"mark,unmark" desc:"Toggle the marked state."`
	Visual          bool   `opt:"-v" aliases:"mark" desc:"Enter / leave visual mark mode."`
	VisualClear     bool   `opt:"-V" aliases:"mark" desc:"Same as -v but does not clear existing selection."`
	Thread          bool   `opt:"-T" aliases:"mark,unmark" desc:"Mark messages from the selected thread."`
	SenderFilter    bool   `opt:"-s" aliases:"mark,unmark" desc:"Mark messages having the substring in their From: header."`
	RecipientFilter bool   `opt:"-r" aliases:"mark,unmark" desc:"Mark messages having the substring in their To:, Cc:, or Bcc: header."`
	Unread          bool   `opt:"-u" aliases:"mark,unmark" desc:"Mark unread messages"`
	NotUnread       bool   `opt:"-U" aliases:"mark,unmark" desc:"Mark read messages"`
	FilterString    string `opt:"..." required:"false" desc:"Mark messages matching this string."`
}

func init() {
	commands.Register(Mark{})
}

func (Mark) Description() string {
	return "Mark, unmark or remark messages."
}

func (Mark) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (Mark) Aliases() []string {
	return []string{"mark", "unmark", "remark"}
}

func (m Mark) Execute(args []string) error {
	h := newHelper()

	store, err := h.store()
	if err != nil {
		return err
	}
	marker := store.Marker()

	if m.Thread && m.All {
		return fmt.Errorf("-a and -T are mutually exclusive")
	}
	if m.Thread && (m.Visual || m.VisualClear) {
		return fmt.Errorf("-v and -T are mutually exclusive")
	}
	if m.Visual && m.All {
		return fmt.Errorf("-a and -v are mutually exclusive")
	}
	if m.SenderFilter && m.RecipientFilter {
		return fmt.Errorf("-s and -r are mutually exclusive")
	}
	if m.Visual && m.FilterString != "" {
		return fmt.Errorf("visual mode does not support filtering")
	}
	if m.Unread == m.NotUnread && m.Unread {
		return fmt.Errorf("-u and -U are mutually exclusive")
	}
	if (m.SenderFilter || m.RecipientFilter) && m.FilterString == "" {
		return fmt.Errorf("-s and -r require a filter string")
	}

	// if filtering and only a single message is provided,
	m.All = (m.FilterString != "" && !(m.Thread || m.All)) ||
		// or if filtering by read status,
		((m.Unread || m.NotUnread) && !(m.Thread || m.All)) ||
		// filter all instead
		m.All

	// fallback: selected message
	filter := func(yield func(models.UID) bool) {
		selected, err := h.msgProvider.SelectedMessage()
		if err != nil {
			log.Errorf("failed to retrieve selected message: %v", err)
			return
		}
		yield(selected.Uid)
	}

	switch {
	case m.All:
		filter = slices.Values(store.Uids())
	case m.Thread:
		threadPtr, err := store.SelectedThread()
		if err != nil {
			return err
		}
		filter = slices.Values(threadPtr.Root().Uids())
	}

	switch {
	case m.SenderFilter:
		filter = senderFilter(store, m.FilterString)(filter)
	case m.RecipientFilter:
		filter = recipientFilter(store, m.FilterString)(filter)
	case m.FilterString != "":
		filter = subjectFilter(store, m.FilterString)(filter)
	}
	if m.Unread {
		filter = seenFilter(store, false)(filter)
	}
	if m.NotUnread {
		filter = seenFilter(store, true)(filter)
	}

	switch args[0] {
	case "mark":
		var modFunc func(models.UID)
		if m.Toggle {
			modFunc = marker.ToggleMark
		} else {
			modFunc = marker.Mark
		}

		if m.Visual || m.VisualClear {
			marker.ToggleVisualMark(m.VisualClear)
			return nil
		}

		for uid := range filter {
			modFunc(uid)
		}
		return nil

	case "unmark":
		if m.Visual || m.VisualClear {
			return fmt.Errorf("visual mode not supported for this command")
		}

		var modFunc func(models.UID)
		if m.Toggle {
			modFunc = marker.ToggleMark
		} else {
			modFunc = marker.Unmark
		}

		for uid := range filter {
			modFunc(uid)
		}
		return nil
	case "remark":
		marker.Remark()
		return nil
	}
	return nil // never reached
}

type filterFunc func(iter.Seq[models.UID]) iter.Seq[models.UID]

func senderFilter(store *lib.MessageStore, senderMatches string) filterFunc {
	return func(uids iter.Seq[models.UID]) iter.Seq[models.UID] {
		return func(yield func(models.UID) bool) {
			for uid := range uids {
				msg := store.Messages[uid]
				if msg == nil || msg.Envelope == nil {
					continue
				}
				from := msg.Envelope.From
				for _, sender := range from {
					if strings.Contains(sender.String(), senderMatches) {
						if !yield(uid) {
							return
						}
						break
					}
				}
			}
		}
	}
}

func recipientFilter(store *lib.MessageStore, recipientMatches string) filterFunc {
	return func(uids iter.Seq[models.UID]) iter.Seq[models.UID] {
		return func(yield func(models.UID) bool) {
			for uid := range uids {
				msg := store.Messages[uid]
				if msg == nil {
					continue
				}
				recipients := slices.Concat(msg.Envelope.To, msg.Envelope.Cc, msg.Envelope.Bcc)
				for _, recipient := range recipients {
					if strings.Contains(recipient.String(), recipientMatches) {
						if !yield(uid) {
							return
						}
						break
					}
				}
			}
		}
	}
}

func subjectFilter(store *lib.MessageStore, subjectMatches string) filterFunc {
	return func(uids iter.Seq[models.UID]) iter.Seq[models.UID] {
		return func(yield func(models.UID) bool) {
			for uid := range uids {
				msg := store.Messages[uid]
				if msg == nil {
					continue
				}
				subject := msg.Envelope.Subject
				if strings.Contains(subject, subjectMatches) {
					if !yield(uid) {
						return
					}
				}
			}
		}
	}
}

func seenFilter(store *lib.MessageStore, isSeen bool) filterFunc {
	return func(uids iter.Seq[models.UID]) iter.Seq[models.UID] {
		return func(yield func(models.UID) bool) {
			for uid := range uids {
				msg := store.Messages[uid]
				if msg == nil {
					continue
				}
				if msg.Flags.Has(models.SeenFlag) == isSeen {
					if !yield(uid) {
						return
					}
				}
			}
		}
	}
}
