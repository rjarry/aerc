aerc-search(1)

# IMAP

*search* [-ru] <terms...>
	Searches the current folder for <terms>. Each separate term is searched
	case-insensitively among subject lines.

	*-r*: Search for read messages

	*-u*: Search for unread messages

# NOTMUCH

*search* <query...>
	You can use the full notmuch query language as described in
	*notmuch-search-terms*(7).

	The query will only apply on top of the active folder query.

	Example: jump to next unread `:search tag:unread`
