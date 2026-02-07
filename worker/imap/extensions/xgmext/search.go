package xgmext

import "github.com/emersion/go-imap"

type ThreadIDSearch struct {
	Charset   string
	ThreadIDs []string
}

// NewThreadIDSearch return an imap.Command to search UIDs for the provided
// thread IDs using the X-GM-EXT-1 (Gmail extension)
func NewThreadIDSearch(threadIDs []string) *ThreadIDSearch {
	return &ThreadIDSearch{
		Charset:   "UTF-8",
		ThreadIDs: threadIDs,
	}
}

func (cmd *ThreadIDSearch) Command() *imap.Command {
	const threadSearchKey = "X-GM-THRID"

	var args []any
	if cmd.Charset != "" {
		args = append(args, imap.RawString("CHARSET"))
		args = append(args, imap.RawString(cmd.Charset))
	}

	// we want to produce a search query that looks like this:
	// SEARCH CHARSET UTF-8 OR OR X-GM-THRID 1771431779961568536 \
	// X-GM-THRID 1765355745646219617 X-GM-THRID 1771500774375286796
	for i := 0; i < len(cmd.ThreadIDs)-1; i++ {
		args = append(args, imap.RawString("OR"))
	}

	for _, thrid := range cmd.ThreadIDs {
		args = append(args, imap.RawString(threadSearchKey))
		args = append(args, imap.RawString(thrid))
	}

	return &imap.Command{
		Name:      "SEARCH",
		Arguments: args,
	}
}

type RawSearch struct {
	Charset string
	Search  string
}

func NewRawSearch(search string) *RawSearch {
	return &RawSearch{
		Charset: "UTF-8",
		Search:  search,
	}
}

func (cmd *RawSearch) Command() *imap.Command {
	const key = "X-GM-RAW"

	var args []any
	if cmd.Charset != "" {
		args = append(args, imap.RawString("CHARSET"))
		args = append(args, imap.RawString(cmd.Charset))
	}

	args = append(args, imap.RawString(key))
	args = append(args, imap.RawString(cmd.Search))

	return &imap.Command{
		Name:      "SEARCH",
		Arguments: args,
	}
}
