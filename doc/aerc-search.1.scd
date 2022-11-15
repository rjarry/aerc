AERC-SEARCH(1)

# NAME

aerc-search - search and filter patterns and options for *aerc*(1)

# MAILDIR & IMAP

*search* [*-ruba*] [*-x* _<flag>_] [*-X* _<flag>_] [*-f* _<from>_] [*-t* _<to>_] [*-c* _<cc>_] [*-d* _<start[,end]>_] [_<terms>_...]
	Searches the current folder for messages matching the given set of
	conditions.

	Each space separated term of _<terms>_, if provided, is searched
	case-insensitively among subject lines unless *-b* or *-a* are
	provided.

	*-r*: Search for read messages

	*-u*: Search for unread messages

	*-x* _<flag>_, *-X* _<flag>_: Restrict search to messages with or without _<flag>_
		Use *-x* to search for messages with the flag set.
		Use *-X* to search for messages without the flag set.

		Possible values are:
			_Seen_
				Read messages
			_Answered_
				Replied messages
			_Flagged_
				Flagged messages

	*-b*: Search in the body of the messages

	*-a*: Search in the entire text of the messages

	*-f* _<from>_: Search for messages from _<from>_

	*-t* _<to>_: Search for messages to _<to>_

	*-c* _<cc>_: Search for messages cc'ed to _<cc>_

	*-d* _<start[..end]>_:
		Search for messages within a particular date range defined as
		\[start, end) where the dates are in the YYYY-MM-DD format.

# NOTMUCH

*search* _query_...
	You can use the full notmuch query language as described in
	*notmuch-search-terms*(7).

	The query will only apply on top of the active folder query.

	Example, jump to next unread:

		:search tag:unread

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
