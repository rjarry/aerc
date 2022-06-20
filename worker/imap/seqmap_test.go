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

	assert.Equal(seqmap.Size(), 0)

	_, found = seqmap.Get(42)
	assert.Equal(found, false)

	_, found = seqmap.Pop(0)
	assert.Equal(found, false)

	seqmap.Put(1, 1337)
	seqmap.Put(2, 42)
	seqmap.Put(3, 1107)
	assert.Equal(seqmap.Size(), 3)

	_, found = seqmap.Pop(0)
	assert.Equal(found, false)

	uid, found = seqmap.Get(1)
	assert.Equal(uid, uint32(1337))
	assert.Equal(found, true)

	uid, found = seqmap.Pop(1)
	assert.Equal(uid, uint32(1337))
	assert.Equal(found, true)
	assert.Equal(seqmap.Size(), 2)

	_, found = seqmap.Pop(1)
	assert.Equal(found, false)
	assert.Equal(seqmap.Size(), 2)

	seqmap.Put(1, 7331)
	assert.Equal(seqmap.Size(), 3)

	seqmap.Clear()
	assert.Equal(seqmap.Size(), 0)

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

	assert.Equal(seqmap.Size(), 0)
}
