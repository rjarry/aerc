package imap

import (
	"sync"
)

// Provider dependent EXPUNGE handler
//
// To delete N items in a single command, aerc does the following:
//   1. Set the \Deleted flag on those N items (identified by their sequence
//      number)
//   2. Call the EXPUNGE command
// It then gets N ExpungeData messages from go-imap, that reference the
// individual message actually deleted.
// Unfortunately the IMAP RFC does not specify the order in which those
// messages should be sent, and different providers have different policies.
// In particular:
//   - GMail and FastMail delete messages by increasing sequence number, and
//     at each individual delete, decrement the sequence number of all the
//     messages that still need to be deleted.
//   - Office 365 deletes messages by decreasing sequence number.
//   - Dovecot deletes messages in a seemingly random order.
// The role of ExpungeHandler is to abstract out those differences, and
// automatically adapt to the IMAP server's behaviour.
// Since there's a non-zero probability that the automatic detection is wrong
// if the server deletes items in a random order, it's also possible to
// statically configure the expunge policy in accounts.conf.

// The IMAP server behaviour when deleting multiple messages.
const (
	// Automatically detect behaviour from the first reply.
	ExpungePolicyAuto = iota
	// The server deletes message in increasing sequence number. After each
	// delete, outstanding messages need to have their sequence numbers
	// decremented.
	ExpungePolicyLowToHigh
	// The server deletes messages in any order, but does not change any of the
	// sequence numbers.
	ExpungePolicyStable
)

type ExpungeHandler struct {
	lock     sync.Mutex
	worker   *IMAPWorker
	policy   int
	items    map[uint32]uint32
	minNum   uint32
	gotFirst bool
}

// Create a new ExpungeHandler for a list of UIDs that are being deleted or
// moved.
func NewExpungeHandler(worker *IMAPWorker, uids []uint32) *ExpungeHandler {
	snapshot, min := worker.seqMap.Snapshot(uids)
	return &ExpungeHandler{
		worker:   worker,
		policy:   worker.config.expungePolicy,
		items:    snapshot,
		minNum:   min,
		gotFirst: false,
	}
}

// Translate the sequence number received from the IMAP server into the
// associated UID, deduce the policy used by the server from the first reply,
// and update the remaining mappings according to that policy if required.
func (h *ExpungeHandler) PopSequenceNumber(seqNum uint32) (uint32, bool) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if !h.gotFirst {
		h.gotFirst = true
		logPrefix := "Configured"
		// This is the very first reply we get; use it to infer the IMAP
		// server's policy if the configuration asks us to.
		if h.policy == ExpungePolicyAuto {
			logPrefix = "Deduced"
			if seqNum == h.minNum {
				h.policy = ExpungePolicyLowToHigh
			} else {
				h.policy = ExpungePolicyStable
			}
		}
		switch h.policy {
		case ExpungePolicyLowToHigh:
			h.worker.worker.Debugf("%s expunge policy: low-to-high", logPrefix)
		case ExpungePolicyStable:
			h.worker.worker.Debugf("%s expunge policy: stable", logPrefix)
		}
	}
	// Resolve the UID from the sequence number and pop the expunger entry.
	uid, ok := h.items[seqNum]
	delete(h.items, seqNum)

	// If the server uses the "low to high" policy, we need to decrement all
	// the remaining entries since the server is doing the same on its end.
	if ok && h.policy == ExpungePolicyLowToHigh {
		newSeq := make(map[uint32]uint32)

		for s, uid := range h.items {
			newSeq[s-1] = uid
		}

		h.items = newSeq
	}

	if !ok {
		h.worker.worker.Errorf("Unexpected sequence number; consider" +
			"overriding the expunge-policy IMAP configuration")
	}

	return uid, ok
}

func (h *ExpungeHandler) IsExpunging(uid uint32) bool {
	h.lock.Lock()
	defer h.lock.Unlock()
	for _, u := range h.items {
		if u == uid {
			return true
		}
	}
	return false
}
