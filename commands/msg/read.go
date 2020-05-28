package msg

import (
	"errors"
	"sync"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Read struct{}

func init() {
	register(Read{})
}

func (Read) Aliases() []string {
	return []string{"read", "unread"}
}

func (Read) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Read) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "t")
	if err != nil {
		return err
	}
	if optind != len(args) {
		return errors.New("Usage: " + args[0] + " [-t]")
	}
	var toggle bool

	for _, opt := range opts {
		switch opt.Option {
		case 't':
			toggle = true
		}
	}

	h := newHelper(aerc)
	store, err := h.store()
	if err != nil {
		return err
	}

	if toggle {
		// ignore command given, simply toggle all the read states
		return submitToggle(aerc, store, h)
	}
	msgUids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}
	switch args[0] {
	case "read":
		submitReadChange(aerc, store, msgUids, true)
	case "unread":
		submitReadChange(aerc, store, msgUids, false)

	}
	return nil
}

func splitMessages(msgs []*models.MessageInfo) (read []uint32, unread []uint32) {
	for _, m := range msgs {
		var seen bool
		for _, flag := range m.Flags {
			if flag == models.SeenFlag {
				seen = true
				break
			}
		}
		if seen {
			read = append(read, m.Uid)
		} else {
			unread = append(unread, m.Uid)
		}
	}
	return read, unread
}

func submitReadChange(aerc *widgets.Aerc, store *lib.MessageStore,
	uids []uint32, newState bool) {
	store.Read(uids, newState, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus(msg_success, 10*time.Second)
		case *types.Error:
			aerc.PushError(" " + msg.Error.Error())
		}
	})
}

func submitReadChangeWg(aerc *widgets.Aerc, store *lib.MessageStore,
	uids []uint32, newState bool, wg *sync.WaitGroup, success *bool) {
	store.Read(uids, newState, func(msg types.WorkerMessage) {
		wg.Add(1)
		switch msg := msg.(type) {
		case *types.Done:
			wg.Done()
		case *types.Error:
			aerc.PushError(" " + msg.Error.Error())
			*success = false
			wg.Done()
		}
	})
}

func submitToggle(aerc *widgets.Aerc, store *lib.MessageStore, h *helper) error {
	msgs, err := h.messages()
	if err != nil {
		return err
	}
	read, unread := splitMessages(msgs)

	var wg sync.WaitGroup
	success := true

	if len(read) != 0 {
		newState := false
		submitReadChangeWg(aerc, store, read, newState, &wg, &success)
	}

	if len(unread) != 0 {
		newState := true
		submitReadChangeWg(aerc, store, unread, newState, &wg, &success)
	}
	// we need to do that in the background, else we block the main thread
	go func() {
		wg.Wait()
		if success {
			aerc.PushStatus(msg_success, 10*time.Second)
		}
	}()
	return nil

}

const msg_success = "read state set successfully"
