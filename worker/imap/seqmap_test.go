package imap

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSeqMap(t *testing.T) {
	var seqmap SeqMap
	var uid uint32
	var found bool
	assert := assert.New(t)

	assert.Equal(0, seqmap.Size())

	_, found = seqmap.Get(42)
	assert.Equal(false, found)

	_, found = seqmap.Pop(0)
	assert.Equal(false, found)

	seqmap.Put(1, 1337)
	seqmap.Put(2, 42)
	seqmap.Put(3, 1107)
	assert.Equal(3, seqmap.Size())

	_, found = seqmap.Pop(0)
	assert.Equal(false, found)

	uid, found = seqmap.Get(1)
	assert.Equal(uint32(1337), uid)
	assert.Equal(true, found)

	uid, found = seqmap.Pop(1)
	assert.Equal(uint32(1337), uid)
	assert.Equal(true, found)
	assert.Equal(2, seqmap.Size())

	// Repop the same seqnum should work because of the syncing
	_, found = seqmap.Pop(1)
	assert.Equal(true, found)
	assert.Equal(1, seqmap.Size())

	// sync means we already have a 1. This is replacing that UID so the size
	// shouldn't increase
	seqmap.Put(1, 7331)
	assert.Equal(1, seqmap.Size())

	seqmap.Clear()
	assert.Equal(0, seqmap.Size())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		seqmap.Put(42, 1337)
		time.Sleep(20 * time.Millisecond)
		seqmap.Put(43, 1107)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, found := seqmap.Pop(43); !found; _, found = seqmap.Pop(43) {
			time.Sleep(1 * time.Millisecond)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, found := seqmap.Pop(42); !found; _, found = seqmap.Pop(42) {
			time.Sleep(1 * time.Millisecond)
		}
	}()
	wg.Wait()

	assert.Equal(0, seqmap.Size())
}
