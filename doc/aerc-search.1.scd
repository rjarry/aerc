aerc-search(1)

# IMAP

*search* [-rubt] [-f <from>] <terms...>
	Searches the current folder for <terms>. Each separate term is searched
	case-insensitively among subject lines if *-b* or *-t* are not provided.

	*-r*: Search for read messages

	*-u*: Search for unread messages

	*-b*: Search in the body of the messages

	*-t*: Search in the entire text of the messages

	*-f <from>*: Search for messages from <from>

# NOTMUCH

*search* <query...>
	You can use the full notmuch query language as described in
	*notmuch-search-terms*(7).

	The query will only apply on top of the active folder query.

	Example: jump to next unread `:search tag:unread`
