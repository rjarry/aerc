package iterator

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

// defaultFactory
type defaultFactory struct{}

func (df *defaultFactory) NewIterator(a interface{}) Iterator {
	switch data := a.(type) {
	case []uint32:
		return &defaultUid{data: data, index: len(data)}
	case []*types.Thread:
		return &defaultThread{data: data, index: len(data)}
	}
	panic(errors.New("a iterator for this type is not implemented yet"))
}

// defaultUid
type defaultUid struct {
	data  []uint32
	index int
}

func (du *defaultUid) Next() bool {
	du.index--
	return du.index >= 0
}

func (du *defaultUid) Value() interface{} {
	return du.data[du.index]
}

func (du *defaultUid) StartIndex() int {
	return len(du.data) - 1
}

func (du *defaultUid) EndIndex() int {
	return 0
}

// defaultThread
type defaultThread struct {
	data  []*types.Thread
	index int
}

func (dt *defaultThread) Next() bool {
	dt.index--
	return dt.index >= 0
}

func (dt *defaultThread) Value() interface{} {
	return dt.data[dt.index]
}

func (dt *defaultThread) StartIndex() int {
	return len(dt.data) - 1
}

func (dt *defaultThread) EndIndex() int {
	return 0
}

// reverseFactory
type reverseFactory struct{}

func (rf *reverseFactory) NewIterator(a interface{}) Iterator {
	switch data := a.(type) {
	case []uint32:
		return &reverseUid{data: data, index: -1}
	case []*types.Thread:
		return &reverseThread{data: data, index: -1}
	}
	panic(errors.New("an iterator for this type is not implemented yet"))
}

// reverseUid
type reverseUid struct {
	data  []uint32
	index int
}

func (ru *reverseUid) Next() bool {
	ru.index++
	return ru.index < len(ru.data)
}

func (ru *reverseUid) Value() interface{} {
	return ru.data[ru.index]
}

func (ru *reverseUid) StartIndex() int {
	return 0
}

func (ru *reverseUid) EndIndex() int {
	return len(ru.data) - 1
}

// reverseThread
type reverseThread struct {
	data  []*types.Thread
	index int
}

func (rt *reverseThread) Next() bool {
	rt.index++
	return rt.index < len(rt.data)
}

func (rt *reverseThread) Value() interface{} {
	return rt.data[rt.index]
}

func (rt *reverseThread) StartIndex() int {
	return 0
}

func (rt *reverseThread) EndIndex() int {
	return len(rt.data) - 1
}
