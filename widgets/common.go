package widgets

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
)

func msgInfoFromUids(store *lib.MessageStore, uids []uint32) ([]*models.MessageInfo, error) {
	infos := make([]*models.MessageInfo, len(uids))
	for i, uid := range uids {
		var ok bool
		infos[i], ok = store.Messages[uid]
		if !ok {
			return nil, fmt.Errorf("uid not found")
		}
	}
	return infos, nil
}
