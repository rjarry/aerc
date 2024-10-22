package cache

import (
	"reflect"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
)

type FolderContents struct {
	MailboxID  jmap.ID
	QueryState string
	Filter     *types.SearchCriteria
	Sort       []*types.SortCriterion
	MessageIDs []jmap.ID
}

func (c *JMAPCache) GetFolderContents(mailboxId jmap.ID) (*FolderContents, error) {
	key := folderContentsKey(mailboxId)
	buf, err := c.get(key)
	if err != nil {
		return nil, err
	}
	m := new(FolderContents)
	err = unmarshal(buf, m)
	if err != nil {
		log.Debugf("cache format has changed, purging foldercontents")
		if e := c.purge("foldercontents/"); e != nil {
			log.Errorf("foldercontents cache purge: %s", e)
		}
		return nil, err
	}
	return m, nil
}

func (c *JMAPCache) PutFolderContents(mailboxId jmap.ID, m *FolderContents) error {
	buf, err := marshal(m)
	if err != nil {
		return err
	}
	return c.put(folderContentsKey(mailboxId), buf)
}

func (c *JMAPCache) DeleteFolderContents(mailboxId jmap.ID) error {
	return c.delete(folderContentsKey(mailboxId))
}

func folderContentsKey(mailboxId jmap.ID) string {
	return "foldercontents/" + string(mailboxId)
}

func (f *FolderContents) NeedsRefresh(
	filter *types.SearchCriteria, sort []*types.SortCriterion,
) bool {
	return f.QueryState == "" ||
		!reflect.DeepEqual(f.Sort, sort) ||
		!reflect.DeepEqual(f.Filter, filter)
}
