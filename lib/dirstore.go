package lib

type DirStore struct {
	dirs []string
}

func NewDirStore() *DirStore {
	return &DirStore{}
}

func (store *DirStore) Update(dirs []string) {
	store.dirs = make([]string, len(dirs))
	copy(store.dirs, dirs)
}

func (store *DirStore) List() []string {
	return store.dirs
}
