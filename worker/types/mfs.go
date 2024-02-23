package types

// MultiFileStrategy represents a strategy for taking file-based actions (e.g.,
// move, copy, delete) on messages that are represented by more than one file.
// These strategies are only used by the notmuch backend but are defined in this
// package to prevent import cycles.
type MultiFileStrategy uint

const (
	Refuse MultiFileStrategy = iota
	ActAll
	ActOne
	ActOneDelRest
	ActDir
	ActDirDelRest
)

var StrToStrategy = map[string]MultiFileStrategy{
	"refuse":              Refuse,
	"act-all":             ActAll,
	"act-one":             ActOne,
	"act-one-delete-rest": ActOneDelRest,
	"act-dir":             ActDir,
	"act-dir-delete-rest": ActDirDelRest,
}

func StrategyStrs() []string {
	strs := make([]string, len(StrToStrategy))
	for s := range StrToStrategy {
		strs = append(strs, s)
	}
	return strs
}
