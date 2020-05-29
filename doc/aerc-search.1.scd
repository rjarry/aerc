aerc-search(1)

# IMAP

*search* [-ruba] [-f <from>] [-t <to>] [-c <cc>] [terms...]
	Searches the current folder for messages matching the given set of
	conditions.

	Each space separated term of *terms*, if provided, is searched
	case-insensitively among subject lines unless *-b* or *-a* are
	provided.

	*-r*: Search for read messages

	*-u*: Search for unread messages

	*-b*: Search in the body of the messages

	*-a*: Search in the entire text of the messages

	*-f <from>*: Search for messages from <from>

	*-t <to>*: Search for messages to <to>

	*-c <cc>*: Search for messages cc'ed to <cc>

# MAILDIR

*search* [-ruba] [-f <from>] [-t <to>] [-c <cc>] [terms...]
	Searches the current folder for messages matching the given set of
	conditions.

	Each space separated term of *terms*, if provided, is searched
	case-insensitively among subject lines unless *-b* or *-a* are
	provided.

	*-r*: Search for read messages

	*-u*: Search for unread messages

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
