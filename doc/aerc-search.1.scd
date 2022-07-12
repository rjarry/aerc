aerc-search(1)

# NAME

aerc-search - search and filter patterns and options for *aerc*(1)

# IMAP

*search* [-ruba] [-x <flag>] [-X <flag>] [-f <from>] [-t <to>] [-c <cc>] [terms...]
	Searches the current folder for messages matching the given set of
	conditions.

	Each space separated term of *terms*, if provided, is searched
	case-insensitively among subject lines unless *-b* or *-a* are
	provided.

	*-r*: Search for read messages

	*-u*: Search for unread messages

	*-x <flag>*, *-X <flag>*: Restrict search to messages with or without <flag>

	Use *-x* to search for messages with the flag set.
	Use *-X* to search for messages without the flag set.

	Possible values are:

	Seen
		Read messages

	Answered
		Replied messages

	Flagged
		Flagged messages

	*-b*: Search in the body of the messages

	*-a*: Search in the entire text of the messages

	*-f <from>*: Search for messages from <from>

	*-t <to>*: Search for messages to <to>

	*-c <cc>*: Search for messages cc'ed to <cc>

# MAILDIR

*search* [-ruba] [-x <flag>] [-X <flag>] [-f <from>] [-t <to>] [-c <cc>] [terms...]
	Searches the current folder for messages matching the given set of
	conditions.

	Each space separated term of *terms*, if provided, is searched
	case-insensitively among subject lines unless *-b* or *-a* are
	provided.

	*-r*: Search for read messages

	*-u*: Search for unread messages

	*-x <flag>*, *-X <flag>*: Restrict search to messages with or without <flag>

	Use *-x* to search for messages with the flag set.
	Use *-X* to search for messages without the flag set.

	Possible values are:

	Seen
		Read messages

	Answered
		Replied messages

	Flagged
		Flagged messages

	*-b*: Search in the body of the messages

	*-a*: Search in the entire text of the messages

	*-f <from>*: Search for messages from <from>

	*-t <to>*: Search for messages to <to>

	*-c <cc>*: Search for messages cc'ed to <cc>

# NOTMUCH

*search* <query...>
	You can use the full notmuch query language as described in
	*notmuch-search-terms*(7).

	The query will only apply on top of the active folder query.

	Example: jump to next unread `:search tag:unread`

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
