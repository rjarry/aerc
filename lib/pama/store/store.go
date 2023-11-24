package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	keyPrefix = "project."
)

var (
	// versTag should be incremented when the underyling data structure
	// changes.
	versTag    = []byte("0001")
	versTagKey = []byte("version.tag")
	currentKey = []byte("current.project")
)

func createKey(name string) []byte {
	return []byte(keyPrefix + name)
}

func parseKey(key []byte) string {
	return strings.TrimPrefix(string(key), keyPrefix)
}

func isProjectKey(key []byte) bool {
	return bytes.HasPrefix(key, []byte(keyPrefix))
}

func cacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = xdg.ExpandHome("~/.cache")
	}
	return path.Join(dir, "aerc"), nil
}

func openStorage() (*leveldb.DB, error) {
	cd, err := cacheDir()
	if err != nil {
		return nil, fmt.Errorf("Unable to find project store directory: %w", err)
	}
	p := path.Join(cd, "projects")

	db, err := leveldb.OpenFile(p, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to open project store: %w", err)
	}

	has, err := db.Has(versTagKey, nil)
	if err != nil {
		return nil, err
	}
	setTag := !has
	if has {
		vers, err := db.Get(versTagKey, nil)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(vers, versTag) {
			log.Warnf("patch store: version mismatch: wipe data")
			iter := db.NewIterator(nil, nil)
			for iter.Next() {
				err := db.Delete(iter.Key(), nil)
				if err != nil {
					log.Errorf("delete: %v")
				}
			}
			iter.Release()
			setTag = true
		}
	}

	if setTag {
		err := db.Put(versTagKey, versTag, nil)
		if err != nil {
			return nil, err
		}
		log.Infof("patch store: set version: %s", string(versTag))
	}

	return db, nil
}

func encode(p models.Project) ([]byte, error) {
	return json.Marshal(p)
}

func decode(data []byte) (p models.Project, err error) {
	err = json.Unmarshal(data, &p)
	return
}

func Store() models.PersistentStorer {
	return &instance{}
}

type instance struct{}

func (instance) StoreProject(p models.Project, overwrite bool) error {
	db, err := openStorage()
	if err != nil {
		return err
	}
	defer db.Close()

	key := createKey(p.Name)
	has, err := db.Has(key, nil)
	if err != nil {
		return err
	}
	if has && !overwrite {
		return fmt.Errorf("Project '%s' already exists.", p.Name)
	}

	log.Debugf("project data: %v", p)

	encoded, err := encode(p)
	if err != nil {
		return err
	}
	return db.Put(key, encoded, nil)
}

func (instance) DeleteProject(name string) error {
	db, err := openStorage()
	if err != nil {
		return err
	}
	defer db.Close()

	key := createKey(name)
	has, err := db.Has(key, nil)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("Project does not exist.")
	}
	return db.Delete(key, nil)
}

func (instance) CurrentName() (string, error) {
	db, err := openStorage()
	if err != nil {
		return "", err
	}
	defer db.Close()
	cur, err := db.Get(currentKey, nil)
	if err != nil {
		return "", err
	}
	return parseKey(cur), nil
}

func (instance) SetCurrent(name string) error {
	db, err := openStorage()
	if err != nil {
		return err
	}
	defer db.Close()
	key := createKey(name)
	return db.Put(currentKey, key, nil)
}

func (instance) Current() (models.Project, error) {
	db, err := openStorage()
	if err != nil {
		return models.Project{}, err
	}
	defer db.Close()

	has, err := db.Has(currentKey, nil)
	if err != nil {
		return models.Project{}, err
	}
	if !has {
		return models.Project{}, fmt.Errorf("No (current) project found; run 'project init' first.")
	}
	curProjectKey, err := db.Get(currentKey, nil)
	if err != nil {
		return models.Project{}, err
	}
	raw, err := db.Get(curProjectKey, nil)
	if err != nil {
		return models.Project{}, err
	}
	p, err := decode(raw)
	if err != nil {
		return models.Project{}, err
	}
	return p, nil
}

func (instance) Names() ([]string, error) {
	db, err := openStorage()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var names []string
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		if !isProjectKey(iter.Key()) {
			continue
		}
		names = append(names, parseKey(iter.Key()))
	}
	iter.Release()
	return names, nil
}

func (instance) Project(name string) (models.Project, error) {
	db, err := openStorage()
	if err != nil {
		return models.Project{}, err
	}
	defer db.Close()
	raw, err := db.Get(createKey(name), nil)
	if err != nil {
		return models.Project{}, err
	}
	p, err := decode(raw)
	if err != nil {
		return models.Project{}, err
	}
	return p, nil
}

func (instance) Projects() ([]models.Project, error) {
	var projects []models.Project
	db, err := openStorage()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		if !isProjectKey(iter.Key()) {
			continue
		}
		p, err := decode(iter.Value())
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	iter.Release()
	return projects, nil
}
