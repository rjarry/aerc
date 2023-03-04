package parse

import (
	"regexp"
	"sync"

	"git.sr.ht/~rjarry/aerc/log"
)

var reCache sync.Map

// Check if a string matches the specified regular expression.
// The regexp is compiled only once and stored in a cache for future use.
func MatchCache(s, expr string) bool {
	var re interface{}
	var found bool

	if re, found = reCache.Load(expr); !found {
		var err error
		re, err = regexp.Compile(expr)
		if err != nil {
			log.Errorf("`%s` invalid regexp: %s", expr, err)
		}
		reCache.Store(expr, re)
	}
	if re, ok := re.(*regexp.Regexp); ok && re != nil {
		return re.MatchString(s)
	}
	return false
}
