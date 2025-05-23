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

	uids := []uint32{1337, 42, 1107}
	seqmap.Initialize(uids)
	assert.Equal(3, seqmap.Size())
	// Original list should remain unsorted
	assert.Equal([]uint32{1337, 42, 1107}, uids)

	_, found = seqmap.Pop(0)
	assert.Equal(false, found)

	uid, found = seqmap.Get(1)
	assert.Equal(42, int(uid))
	assert.Equal(true, found)

	uid, found = seqmap.Pop(1)
	assert.Equal(42, int(uid))
	assert.Equal(true, found)
	assert.Equal(2, seqmap.Size())

	uid, found = seqmap.Get(1)
	assert.Equal(1107, int(uid))

	// Repeated puts of the same UID shouldn't change the size
	seqmap.Put(1231)
	assert.Equal(3, seqmap.Size())
	seqmap.Put(1231)
	assert.Equal(3, seqmap.Size())

	uid, found = seqmap.Get(2)
	assert.Equal(1231, int(uid))

	_, found = seqmap.Pop(1)
	assert.Equal(true, found)
	assert.Equal(2, seqmap.Size())

	seqmap.Initialize(nil)
	assert.Equal(0, seqmap.Size())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		seqmap.Initialize([]uint32{42, 1337})
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, found := seqmap.Pop(1); !found; _, found = seqmap.Pop(1) {
			time.Sleep(1 * time.Millisecond)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, found := seqmap.Pop(1); !found; _, found = seqmap.Pop(1) {
			time.Sleep(1 * time.Millisecond)
		}
	}()
	wg.Wait()

	assert.Equal(0, seqmap.Size())

	//
	// Test snapshotting
	//

	// The sequence is [ 21, 24, 1107, 1982, 2399, 27892, 32000 ] (1 is at
	// position 1, 24 position 2, etc)
	seqmap.Initialize([]uint32{21, 42, 1107, 1982, 2390, 27892, 32000})

	// Snapshot [ 21, 1107, 27892, 1234567 ]; they're all present in the
	// sequence except 27982
	snap, min := seqmap.Snapshot([]uint32{21, 1107, 27892, 1234567})

	// Verify that we did snapshot [ 1 => 21, 3=>1107, 6=>27892 ]
	assert.Equal(3, len(snap))
	assert.Equal(uint32(1), min)
	assert.Equal(snap[1], uint32(21))
	assert.Equal(snap[3], uint32(1107))
	assert.Equal(snap[6], uint32(27892))

	// Vefify that the snapshotted items have been removed from the sequence,
	// that therefore became [ 42, 1982, 2390, 32000 ]
	assert.Equal(4, seqmap.Size())
	o1, _ := seqmap.Get(1)
	assert.Equal(uint32(42), o1)
	o2, _ := seqmap.Get(2)
	assert.Equal(uint32(1982), o2)
	o3, _ := seqmap.Get(3)
	assert.Equal(uint32(2390), o3)
	o4, _ := seqmap.Get(4)
	assert.Equal(uint32(32000), o4)
}
