package lib

type DirStore struct {
	dirs      []string
	msgStores map[string]*MessageStore
}

func NewDirStore() *DirStore {
	msgStores := make(map[string]*MessageStore)
	return &DirStore{msgStores: msgStores}
}

func (store *DirStore) Update(dirs []string) {
	store.dirs = make([]string, len(dirs))
	copy(store.dirs, dirs)
}

func (store *DirStore) List() []string {
	return store.dirs
}

func (store *DirStore) MessageStore(dirname string) (*MessageStore, bool) {
	msgStore, ok := store.msgStores[dirname]
	return msgStore, ok
}

func (store *DirStore) SetMessageStore(name string, msgStore *MessageStore) {
	store.msgStores[name] = msgStore
}
