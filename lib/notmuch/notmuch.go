//go:build notmuch
// +build notmuch

package notmuch

/*
#cgo LDFLAGS: -lnotmuch

#include <stdlib.h>
#include <notmuch.h>

#if !LIBNOTMUCH_CHECK_VERSION(5, 6, 0)
#error "aerc requires libnotmuch.so.5.6 or later"
#endif

*/
import "C"

// NOTE: Any CGO call which passes a reference to a pointer (**object) will fail
// gocritic:dupSubExpr. All of these calls are set to be ignored by the linter
// Reference: https://github.com/go-critic/go-critic/issues/897
