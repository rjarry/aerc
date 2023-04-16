package lib

type DirStore struct {
	msgStores map[string]*MessageStore
}

func NewDirStore() *DirStore {
	msgStores := make(map[string]*MessageStore)
	return &DirStore{msgStores: msgStores}
}

func (store *DirStore) List() []string {
	dirs := []string{}
	for dir := range store.msgStores {
		dirs = append(dirs, dir)
	}
	return dirs
}

func (store *DirStore) MessageStore(dirname string) (*MessageStore, bool) {
	msgStore, ok := store.msgStores[dirname]
	return msgStore, ok
}

func (store *DirStore) SetMessageStore(name string, msgStore *MessageStore) {
	store.msgStores[name] = msgStore
}

func (store *DirStore) Remove(name string) {
	delete(store.msgStores, name)
}
