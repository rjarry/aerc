package widgets

import (
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type MessageStore struct {
	DirInfo  types.DirectoryInfo
	Messages map[uint64]*types.MessageInfo
}

func NewMessageStore(dirInfo *types.DirectoryInfo) *MessageStore {
	return &MessageStore{DirInfo: *dirInfo}
}

func (store *MessageStore) Update(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.DirectoryInfo:
		store.DirInfo = *msg
		break
	case *types.DirectoryContents:
		newMap := make(map[uint64]*types.MessageInfo)
		for _, uid := range msg.Uids {
			if msg, ok := store.Messages[uid]; ok {
				newMap[uid] = msg
			} else {
				newMap[uid] = nil
			}
		}
		store.Messages = newMap
		break
	case *types.MessageInfo:
		store.Messages[msg.Uid] = msg
		break
	}
}
