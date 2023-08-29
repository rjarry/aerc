//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <notmuch.h>

*/
import "C"

type Properties struct {
	key        *C.char
	value      *C.char
	properties *C.notmuch_message_properties_t
}

// Next advances the Properties iterator to the next property. Next returns false if
// no more properties are available
func (p *Properties) Next() bool {
	if C.notmuch_message_properties_valid(p.properties) == 0 {
		return false
	}
	p.key = C.notmuch_message_properties_key(p.properties)
	p.value = C.notmuch_message_properties_value(p.properties)
	C.notmuch_message_properties_move_to_next(p.properties)
	return true
}

// Returns the key of the current iterator location
func (p *Properties) Key() string {
	return C.GoString(p.key)
}

func (p *Properties) Value() string {
	return C.GoString(p.value)
}
