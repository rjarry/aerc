//+build notmuch

package lib

type ThreadNode struct {
	Uid     string
	From    string
	Subject string
	InQuery bool // is the msg included in the query

	Parent      *ThreadNode
	NextSibling *ThreadNode
	FirstChild  *ThreadNode
}
