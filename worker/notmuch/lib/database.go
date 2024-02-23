//go:build notmuch
// +build notmuch

package lib

import (
	"context"
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/notmuch"
	"git.sr.ht/~rjarry/aerc/lib/uidstore"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type DB struct {
	path         string
	excludedTags []string
	db           *notmuch.Database
	uidStore     *uidstore.Store
}

func NewDB(path string, excludedTags []string) *DB {
	nm := &notmuch.Database{
		Path: path,
	}
	db := &DB{
		path:         path,
		excludedTags: excludedTags,
		uidStore:     uidstore.NewStore(),
		db:           nm,
	}
	return db
}

func (db *DB) Connect() error {
	return db.db.Open(notmuch.MODE_READ_ONLY)
}

func (db *DB) Close() error {
	return db.db.Close()
}

// Returns the DB path
func (db *DB) Path() string {
	return db.db.ResolvedPath()
}

// ListTags lists all known tags
func (db *DB) ListTags() []string {
	return db.db.Tags()
}

// State returns the lastmod of the database. This is a uin64 which is
// incremented with every modification
func (db *DB) State() uint64 {
	_, lastmod := db.db.Revision()
	return lastmod
}

// getQuery returns a query based on the provided query string.
// It also configures the query as specified on the worker
func (db *DB) newQuery(query string) (*notmuch.Query, error) {
	q, err := db.db.Query(query)
	if err != nil {
		return nil, err
	}
	q.Exclude(notmuch.EXCLUDE_ALL)
	q.Sort(notmuch.SORT_OLDEST_FIRST)
	for _, t := range db.excludedTags {
		err := q.ExcludeTag(t)
		if err != nil {
			return nil, err
		}
	}
	return &q, nil
}

func (db *DB) MsgIDFromFilename(filename string) (string, error) {
	msg, err := db.db.FindMessageByFilename(filename)
	if err != nil {
		return "", err
	}
	defer msg.Close()
	return msg.ID(), nil
}

func (db *DB) MsgIDsFromQuery(ctx context.Context, q string) ([]string, error) {
	query, err := db.newQuery(q)
	if err != nil {
		return nil, err
	}
	defer query.Close()
	messages, err := query.Messages()
	if err != nil {
		return nil, err
	}
	defer messages.Close()
	var msgIDs []string
	for messages.Next() {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		default:
			msg := messages.Message()
			defer msg.Close()
			msgIDs = append(msgIDs, msg.ID())
		}
	}
	return msgIDs, err
}

func (db *DB) ThreadsFromQuery(ctx context.Context, q string, entireThread bool) ([]*types.Thread, error) {
	query, err := db.newQuery(q)
	if err != nil {
		return nil, err
	}
	defer query.Close()
	// To get proper ordering of threads, we always sort newest first
	query.Sort(notmuch.SORT_NEWEST_FIRST)
	threads, err := query.Threads()
	if err != nil {
		return nil, err
	}
	n, err := query.CountMessages()
	if err != nil {
		return nil, err
	}
	defer threads.Close()
	res := make([]*types.Thread, 0, n)
	for threads.Next() {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		default:
			thread := threads.Thread()
			tlm := thread.TopLevelMessages()
			root := db.makeThread(nil, &tlm, entireThread)
			// if len(root) > 1 {
			// TODO make a dummy root node and link all the
			// first level children to it
			// }
			res = append(res, root...)
			tlm.Close()
			thread.Close()
		}
	}
	// Reverse the slice
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return res, err
}

type MessageCount struct {
	Exists int
	Unread int
}

func (db *DB) QueryCountMessages(q string) (MessageCount, error) {
	count := MessageCount{}
	query, err := db.newQuery(q)
	if err != nil {
		return count, err
	}
	defer query.Close()
	count.Exists, err = query.CountMessages()
	if err != nil {
		return count, err
	}

	unreadQuery, err := db.newQuery(AndQueries(q, "tag:unread"))
	if err != nil {
		return count, err
	}
	defer unreadQuery.Close()
	count.Unread, err = unreadQuery.CountMessages()
	if err != nil {
		return count, err
	}

	return count, nil
}

func (db *DB) MsgFilename(key string) (string, error) {
	msg, err := db.db.FindMessageByID(key)
	if err != nil {
		return "", err
	}
	defer msg.Close()
	return msg.Filename(), nil
}

func (db *DB) MsgTags(key string) ([]string, error) {
	msg, err := db.db.FindMessageByID(key)
	if err != nil {
		return nil, err
	}
	defer msg.Close()
	return msg.Tags(), nil
}

func (db *DB) MsgFilenames(key string) ([]string, error) {
	msg, err := db.db.FindMessageByID(key)
	if err != nil {
		return nil, err
	}
	defer msg.Close()
	return msg.Filenames(), nil
}

func (db *DB) DeleteMessage(filename string) error {
	err := db.db.Reopen(notmuch.MODE_READ_WRITE)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.db.Reopen(notmuch.MODE_READ_ONLY); err != nil {
			log.Errorf("couldn't reopen: %s", err)
		}
	}()
	err = db.db.BeginAtomic()
	if err != nil {
		return err
	}
	defer func() {
		if err := db.db.EndAtomic(); err != nil {
			log.Errorf("couldn't end atomic: %s", err)
		}
	}()
	err = db.db.RemoveFile(filename)
	if err != nil && !errors.Is(err, notmuch.STATUS_DUPLICATE_MESSAGE_ID) {
		return err
	}
	return nil
}

func (db *DB) IndexFile(filename string) (string, error) {
	err := db.db.Reopen(notmuch.MODE_READ_WRITE)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := db.db.Reopen(notmuch.MODE_READ_ONLY); err != nil {
			log.Errorf("couldn't reopen: %s", err)
		}
	}()
	err = db.db.BeginAtomic()
	if err != nil {
		return "", err
	}
	defer func() {
		if err := db.db.EndAtomic(); err != nil {
			log.Errorf("couldn't end atomic: %s", err)
		}
	}()
	msg, err := db.db.IndexFile(filename)
	if err != nil {
		return "", err
	}
	defer msg.Close()
	return msg.ID(), nil
}

func (db *DB) MsgModifyTags(key string, add, remove []string) error {
	err := db.db.Reopen(notmuch.MODE_READ_WRITE)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.db.Reopen(notmuch.MODE_READ_ONLY); err != nil {
			log.Errorf("couldn't reopen: %s", err)
		}
	}()
	err = db.db.BeginAtomic()
	if err != nil {
		return err
	}
	defer func() {
		if err := db.db.EndAtomic(); err != nil {
			log.Errorf("couldn't end atomic: %s", err)
		}
	}()
	msg, err := db.db.FindMessageByID(key)
	if err != nil {
		return err
	}
	defer msg.Close()
	for _, tag := range add {
		err := msg.AddTag(tag)
		if err != nil {
			log.Warnf("failed to add tag: %v", err)
		}
	}
	for _, tag := range remove {
		err := msg.RemoveTag(tag)
		if err != nil {
			log.Warnf("failed to add tag: %v", err)
		}
	}
	return msg.SyncTagsToMaildirFlags()
}

func (db *DB) UidFromKey(key string) uint32 {
	return db.uidStore.GetOrInsert(key)
}

func (db *DB) KeyFromUid(uid uint32) (string, bool) {
	return db.uidStore.GetKey(uid)
}

func (db *DB) makeThread(parent *types.Thread, msgs *notmuch.Messages, threadContext bool) []*types.Thread {
	var siblings []*types.Thread
	for msgs.Next() {
		msg := msgs.Message()
		defer msg.Close()
		msgID := msg.ID()
		match, err := msg.Flag(notmuch.MESSAGE_FLAG_MATCH)
		if err != nil {
			log.Errorf("%s", err)
			continue
		}
		replies := msg.Replies()
		defer replies.Close()
		if !match && !threadContext {
			siblings = append(siblings, db.makeThread(parent, &replies, threadContext)...)
			continue
		}
		node := &types.Thread{
			Uid:    db.uidStore.GetOrInsert(msgID),
			Parent: parent,
		}
		switch threadContext {
		case true:
			node.Context = !match
		default:
			if match {
				node.Hidden = 0
			} else {
				node.Hidden = 1
			}
		}
		if parent != nil && parent.FirstChild == nil {
			parent.FirstChild = node
		}
		siblings = append(siblings, node)
		db.makeThread(node, &replies, threadContext)
	}
	for i := 1; i < len(siblings); i++ {
		siblings[i-1].NextSibling = siblings[i]
	}
	return siblings
}

func AndQueries(q1, q2 string) string {
	if q1 == "" {
		return q2
	}
	if q2 == "" {
		return q1
	}
	if q1 == "*" {
		return q2
	}
	if q2 == "*" {
		return q1
	}
	return fmt.Sprintf("(%s) and (%s)", q1, q2)
}
