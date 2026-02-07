package imap

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/utf7"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// drainUpdates will drain the updates channel. For some operations, the imap
// server will send unilateral messages. If they arrive while another operation
// is in progress, the buffered updates channel can fill up and cause a freeze
// of the entire backend. Avoid this by draining the updates channel and only
// process the Message and Expunge updates.
//
// To stop the draining, close the returned struct.
func (imapw *IMAPWorker) drainUpdates() *drainCloser {
	done := make(chan struct{})
	go func() {
		defer log.PanicHandler()
		for {
			select {
			case update := <-imapw.updates:
				switch update.(type) {
				case *client.MessageUpdate,
					*client.ExpungeUpdate:
					imapw.handleImapUpdate(update)
				}
			case <-done:
				return
			}
		}
	}()
	return &drainCloser{done}
}

type drainCloser struct {
	done chan struct{}
}

func (d *drainCloser) Close() error {
	close(d.done)
	return nil
}

func (imapw *IMAPWorker) handleDeleteMessages(msg *types.DeleteMessages) error {
	drain := imapw.drainUpdates()
	defer drain.Close()

	// Build provider-dependent EXPUNGE handler.
	imapw.BuildExpungeHandler(imapw.UidToUint32List(msg.Uids), true)

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []any{imap.DeletedFlag}
	uids := imapw.UidListToSeqSet(msg.Uids)
	if err := imapw.client.UidStore(uids, item, flags, nil); err != nil {
		return err
	}
	if err := imapw.client.Expunge(nil); err != nil {
		return err
	}
	return nil
}

func (imapw *IMAPWorker) handleAnsweredMessages(msg *types.AnsweredMessages) error {
	item := imap.FormatFlagsOp(imap.AddFlags, false)
	flags := []any{imap.AnsweredFlag}
	if !msg.Answered {
		item = imap.FormatFlagsOp(imap.RemoveFlags, false)
	}
	return imapw.handleStoreOps(msg.Uids, item, flags,
		func(_msg *imap.Message) error {
			systemFlags, keywordFlags := translateImapFlags(_msg.Flags)
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info: &models.MessageInfo{
					Flags:  systemFlags,
					Labels: keywordFlags,
					Uid:    imapw.Uint32ToUid(_msg.Uid),
				},
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleFlagMessages(msg *types.FlagMessages) error {
	flags := []any{flagToImap[msg.Flags]}
	item := imap.FormatFlagsOp(imap.AddFlags, false)
	if !msg.Enable {
		item = imap.FormatFlagsOp(imap.RemoveFlags, false)
	}
	return imapw.handleStoreOps(msg.Uids, item, flags,
		func(_msg *imap.Message) error {
			systemFlags, keywordFlags := translateImapFlags(_msg.Flags)
			imapw.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info: &models.MessageInfo{
					Flags:  systemFlags,
					Labels: keywordFlags,
					Uid:    imapw.Uint32ToUid(_msg.Uid),
				},
				ReplaceFlags: true,
			}, nil)
			return nil
		})
}

func (imapw *IMAPWorker) handleModifyLabels(msg *types.ModifyLabels) error {
	if len(msg.Toggle) > 0 {
		// To toggle, we need to query the current message, and it's
		// not straightforward from here; cowardly bail out.
		return errors.New("label toggling not supported")
	}
	if imapw.config.provider == Proton {
		// If any label removal is requested, make sure that we're on that
		// label's virtual folder (otherwise the removal is silently a no-op)
		for _, l := range msg.Remove {
			if imapw.selected.Name != fmt.Sprintf("Labels/%s", l) {
				return errors.New("Proton labels can only " +
					"be removed from their virtual folder")
			}
		}
	}
	labelOps := map[imap.FlagsOp][]string{
		imap.AddFlags:    msg.Add,
		imap.RemoveFlags: msg.Remove,
	}
	for labelOp, labels := range labelOps {
		if len(labels) == 0 {
			continue
		}
		switch imapw.config.provider {
		case GMail:
			// Per GMail documentation, leverage the STORE command
			// to update labels
			// (https://developers.google.com/workspace/gmail/imap/imap-extensions)
			var item imap.StoreItem
			if labelOp == imap.AddFlags {
				item = "+X-GM-LABELS"
			} else {
				item = "-X-GM-LABELS"
			}
			// Duplicate the label list to avoid funky things to
			// happen, possibly linked to this (?)
			// https://github.com/emersion/go-imap/blob/v1.2.1/client/cmd_selected.go#L195
			labelsAny := []any{}
			utf7Encoder := utf7.Encoding.NewEncoder()
			for _, l := range labels {
				utf7label, _ := utf7Encoder.String(l)
				labelsAny = append(labelsAny, utf7label)
			}
			nop_cb := func(_ *imap.Message) error { return nil }
			return imapw.handleStoreOps(msg.Uids, item, labelsAny, nop_cb)
		case Proton:
			// Per Proton documentation, adding/removing labels
			// is obtained by moving messages to/from label virtual
			// folders
			// (https://proton.me/support/labels-in-bridge#how-to-apply-labels-in-bridge)
			uids := imapw.UidListToSeqSet(msg.Uids)
			var impactedVFolders []string
			for _, l := range labels {
				var destination string
				if labelOp == imap.AddFlags {
					destination = fmt.Sprintf("Labels/%s", l)
					impactedVFolders = append(impactedVFolders,
						destination)
				} else {
					destination = "INBOX"
				}
				if err := imapw.client.UidMove(uids, destination); err != nil {
					return err
				}
			}
			// Refresh the impacted virtual folders;
			impactedVFolders = slices.Compact(impactedVFolders)
			imapw.worker.PostAction(context.TODO(), &types.CheckMail{
				Directories: impactedVFolders,
			}, nil)
		default:
			if len(imapw.selected.PermanentFlags) == 0 || slices.Contains(imapw.selected.PermanentFlags, "\\*") {
				var item imap.StoreItem
				if labelOp == imap.AddFlags {
					item = "+FLAGS"
				} else {
					item = "-FLAGS"
				}
				labelsAny := []any{}
				// This utf7 encoding may or may not be the right thing to do?
				utf7Encoder := utf7.Encoding.NewEncoder()
				for _, l := range labels {
					utf7label, _ := utf7Encoder.String(l)
					labelsAny = append(labelsAny, utf7label)
				}
				refresh_cb := func(_msg *imap.Message) error {
					systemFlags, keywordFlags := translateImapFlags(_msg.Flags)
					imapw.worker.PostMessage(&types.MessageInfo{
						Message: types.RespondTo(msg),
						Info: &models.MessageInfo{
							Flags:  systemFlags,
							Labels: keywordFlags,
							Uid:    imapw.Uint32ToUid(_msg.Uid),
						},
					}, nil)
					return nil
				}
				return imapw.handleStoreOps(msg.Uids, item, labelsAny, refresh_cb)
			} else {
				return errors.New("operation not supported by this imap server")
			}
		}
	}
	return nil
}

func (imapw *IMAPWorker) handleStoreOps(
	uids []models.UID, item imap.StoreItem, flag any,
	procFunc func(*imap.Message) error,
) error {
	messages := make(chan *imap.Message)
	done := make(chan error)

	go func() {
		defer log.PanicHandler()

		var reterr error
		for _msg := range messages {
			err := procFunc(_msg)
			if err != nil {
				if reterr == nil {
					reterr = err
				}
				// drain the channel upon error
				for range messages {
				}
			}
		}
		done <- reterr
	}()

	set := imapw.UidListToSeqSet(uids)
	if err := imapw.client.UidStore(set, item, flag, messages); err != nil {
		return err
	}
	if err := <-done; err != nil {
		return err
	}
	imapw.worker.PostAction(context.TODO(), &types.CheckMail{
		Directories: []string{imapw.selected.Name},
	}, nil)
	return nil
}
