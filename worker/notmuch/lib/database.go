//+build notmuch

package lib

import (
	"fmt"
	"log"
	"time"

	notmuch "github.com/zenhack/go.notmuch"
)

const MAX_DB_AGE time.Duration = 10 * time.Second

type DB struct {
	path         string
	excludedTags []string
	logger       *log.Logger
	lastOpenTime time.Time
	db           *notmuch.DB
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
	// used as sanity check upon initial connect
	err := db.connect(false)
	return err
}

func (db *DB) close() error {
	if db.db == nil {
		return nil
	}
	err := db.db.Close()
	db.db = nil
	return err
}

func (db *DB) connect(writable bool) error {
	var mode notmuch.DBMode = notmuch.DBReadOnly
	if writable {
		mode = notmuch.DBReadWrite
	}
	var err error
	db.db, err = notmuch.Open(db.path, mode)
	if err != nil {
		return fmt.Errorf("could not connect to notmuch db: %v", err)
	}
	db.lastOpenTime = time.Now()
	return nil
}

//withConnection calls callback on the DB object, cleaning up upon return.
//the error returned is from the connection attempt, if not successful,
//or from the callback otherwise.
func (db *DB) withConnection(writable bool, cb func(*notmuch.DB) error) error {
	too_old := time.Now().After(db.lastOpenTime.Add(MAX_DB_AGE))
	if db.db == nil || writable || too_old {
		if cerr := db.close(); cerr != nil {
			db.logger.Printf("failed to close the notmuch db: %v", cerr)
		}
		err := db.connect(writable)
		if err != nil {
			db.logger.Printf("failed to open the notmuch db: %v", err)
			return err
		}
	}
	err := cb(db.db)
	if writable {
		// we need to close to commit the changes, else we block others
		if cerr := db.close(); cerr != nil {
			db.logger.Printf("failed to close the notmuch db: %v", cerr)
		}
	}
	return err
}

// ListTags lists all known tags
func (db *DB) ListTags() ([]string, error) {
	var result []string
	err := db.withConnection(false, func(ndb *notmuch.DB) error {
		tags, err := ndb.Tags()
		if err != nil {
			return err
		}
		defer tags.Close()
		var tag *notmuch.Tag
		for tags.Next(&tag) {
			result = append(result, tag.Value)
		}
		return nil
	})
	return result, err
}

//getQuery returns a query based on the provided query string.
//It also configures the query as specified on the worker
func (db *DB) newQuery(ndb *notmuch.DB, query string) (*notmuch.Query, error) {
	q := ndb.NewQuery(query)
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
	var msgIDs []string
	err := db.withConnection(false, func(ndb *notmuch.DB) error {
		query, err := db.newQuery(ndb, q)
		if err != nil {
			return err
		}
		defer query.Close()
		msgs, err := query.Messages()
		if err != nil {
			return err
		}
		defer msgs.Close()
		var msg *notmuch.Message
		for msgs.Next(&msg) {
			msgIDs = append(msgIDs, msg.ID())
		}
		return nil
	})
	return msgIDs, err
}

type MessageCount struct {
	Exists int
	Unread int
}

func (db *DB) QueryCountMessages(q string) (MessageCount, error) {
	var (
		exists int
		unread int
	)
	err := db.withConnection(false, func(ndb *notmuch.DB) error {
		query, err := db.newQuery(ndb, q)
		if err != nil {
			return err
		}
		exists = query.CountMessages()
		query.Close()
		uq, err := db.newQuery(ndb, fmt.Sprintf("(%v) and (tag:unread)", q))
		if err != nil {
			return err
		}
		defer uq.Close()
		unread = uq.CountMessages()
		return nil
	})
	return MessageCount{
		Exists: exists,
		Unread: unread,
	}, err
}

func (db *DB) MsgFilename(key string) (string, error) {
	var filename string
	err := db.withConnection(false, func(ndb *notmuch.DB) error {
		msg, err := ndb.FindMessage(key)
		if err != nil {
			return err
		}
		defer msg.Close()
		filename = msg.Filename()
		return nil
	})
	return filename, err
}

func (db *DB) MsgTags(key string) ([]string, error) {
	var tags []string
	err := db.withConnection(false, func(ndb *notmuch.DB) error {
		msg, err := ndb.FindMessage(key)
		if err != nil {
			return err
		}
		defer msg.Close()
		ts := msg.Tags()
		defer ts.Close()
		var tag *notmuch.Tag
		for ts.Next(&tag) {
			tags = append(tags, tag.Value)
		}
		return nil
	})
	return tags, err
}

func (db *DB) msgModify(key string,
	cb func(*notmuch.Message) error) error {
	err := db.withConnection(true, func(ndb *notmuch.DB) error {
		msg, err := ndb.FindMessage(key)
		if err != nil {
			return err
		}
		defer msg.Close()

		cb(msg)
		err = msg.TagsToMaildirFlags()
		if err != nil {
			db.logger.Printf("could not sync maildir flags: %v", err)
		}
		return nil
	})
	return err
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
