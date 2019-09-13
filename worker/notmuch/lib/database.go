//+build notmuch

package lib

import (
	"fmt"
	"log"

	notmuch "github.com/zenhack/go.notmuch"
)

type DB struct {
	path         string
	excludedTags []string
	ro           *notmuch.DB
	logger       *log.Logger
}

func NewDB(path string, excludedTags []string,
	logger *log.Logger) *DB {
	db := &DB{
		path:         path,
		excludedTags: excludedTags,
		logger:       logger,
	}
	return db
}

func (db *DB) Connect() error {
	return db.connectRO()
}

// connectRW returns a writable notmuch DB, which needs to be closed to commit
// the changes and to release the DB lock
func (db *DB) connectRW() (*notmuch.DB, error) {
	rw, err := notmuch.Open(db.path, notmuch.DBReadWrite)
	if err != nil {
		return nil, fmt.Errorf("could not connect to notmuch db: %v", err)
	}
	return rw, err
}

// connectRO connects a RO db to the worker
func (db *DB) connectRO() error {
	if db.ro != nil {
		if err := db.ro.Close(); err != nil {
			db.logger.Printf("connectRO: could not close the old db: %v", err)
		}
	}
	var err error
	db.ro, err = notmuch.Open(db.path, notmuch.DBReadOnly)
	if err != nil {
		return fmt.Errorf("could not connect to notmuch db: %v", err)
	}
	return nil
}

//getQuery returns a query based on the provided query string.
//It also configures the query as specified on the worker
func (db *DB) newQuery(query string) (*notmuch.Query, error) {
	if db.ro == nil {
		return nil, fmt.Errorf("not connected to the notmuch db")
	}
	q := db.ro.NewQuery(query)
	q.SetExcludeScheme(notmuch.EXCLUDE_TRUE)
	q.SetSortScheme(notmuch.SORT_OLDEST_FIRST)
	for _, t := range db.excludedTags {
		err := q.AddTagExclude(t)
		if err != nil && err != notmuch.ErrIgnored {
			return nil, err
		}
	}
	return q, nil
}

func (db *DB) MsgIDsFromQuery(q string) ([]string, error) {
	if db.ro == nil {
		return nil, fmt.Errorf("not connected to the notmuch db")
	}
	query, err := db.newQuery(q)
	if err != nil {
		return nil, err
	}
	msgs, err := query.Messages()
	if err != nil {
		return nil, err
	}
	var msg *notmuch.Message
	var msgIDs []string
	for msgs.Next(&msg) {
		msgIDs = append(msgIDs, msg.ID())
	}
	return msgIDs, nil
}

type MessageCount struct {
	Exists int
	Unread int
}

func (db *DB) QueryCountMessages(q string) (MessageCount, error) {
	query, err := db.newQuery(q)
	if err != nil {
		return MessageCount{}, err
	}
	exists := query.CountMessages()
	query.Close()
	uq, err := db.newQuery(fmt.Sprintf("(%v) and (tag:unread)", q))
	if err != nil {
		return MessageCount{}, err
	}
	defer uq.Close()
	unread := uq.CountMessages()
	return MessageCount{
		Exists: exists,
		Unread: unread,
	}, nil
}

func (db *DB) MsgFilename(key string) (string, error) {
	msg, err := db.ro.FindMessage(key)
	if err != nil {
		return "", err
	}
	defer msg.Close()
	return msg.Filename(), nil
}

func (db *DB) MsgTags(key string) ([]string, error) {
	msg, err := db.ro.FindMessage(key)
	if err != nil {
		return nil, err
	}
	defer msg.Close()
	ts := msg.Tags()
	var tags []string
	var tag *notmuch.Tag
	for ts.Next(&tag) {
		tags = append(tags, tag.Value)
	}
	return tags, nil
}

func (db *DB) msgModify(key string,
	cb func(*notmuch.Message) error) error {
	defer db.connectRO()
	db.ro.Close()

	rw, err := db.connectRW()
	if err != nil {
		return err
	}
	defer rw.Close()

	msg, err := rw.FindMessage(key)
	if err != nil {
		return err
	}
	defer msg.Close()

	cb(msg)
	return nil
}

func (db *DB) MsgModifyTags(key string, add, remove []string) error {
	err := db.msgModify(key, func(msg *notmuch.Message) error {
		ierr := msg.Atomic(func(msg *notmuch.Message) {
			for _, t := range add {
				msg.AddTag(t)
			}
			for _, t := range remove {
				msg.RemoveTag(t)
			}
		})
		return ierr
	})
	return err
}

