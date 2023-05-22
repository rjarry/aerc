package lib

import (
	"git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/models"
)

type DirStore struct {
	dirs      map[string]*models.Directory
	msgStores map[string]*MessageStore
	order     []string
}

func NewDirStore() *DirStore {
	return &DirStore{
		dirs:      make(map[string]*models.Directory),
		msgStores: make(map[string]*MessageStore),
	}
}

func (store *DirStore) List() []string {
	dirs := []string{}
	for dir := range store.msgStores {
		dirs = append(dirs, dir)
	}
	sort.SortStringBy(dirs, store.order)
	return dirs
}

func (store *DirStore) MessageStore(dirname string) (*MessageStore, bool) {
	msgStore, ok := store.msgStores[dirname]
	return msgStore, ok
}

func (store *DirStore) SetMessageStore(dir *models.Directory, msgStore *MessageStore) {
	s := dir.Name
	if _, ok := store.dirs[s]; !ok {
		store.order = append(store.order, s)
	}
	store.dirs[dir.Name] = dir
	store.msgStores[dir.Name] = msgStore
}

func (store *DirStore) Remove(name string) {
	delete(store.dirs, name)
	delete(store.msgStores, name)
}

func (store *DirStore) Directory(name string) *models.Directory {
	return store.dirs[name]
}
