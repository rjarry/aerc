AERC-SEARCH(1)

# NAME

aerc-search - search and filter patterns and options for *aerc*(1)

# MAILDIR & IMAP

*:filter* [*-ruba*] [*-x* _<flag>_] [*-X* _<flag>_] [*-H* _Header: Value_] [*-f* _<from>_] [*-t* _<to>_] [*-c* _<cc>_] [*-d* _<start[,end]>_] [_<terms>_...]
*:search* [*-ruba*] [*-x* _<flag>_] [*-X* _<flag>_] [*-H* _Header: Value_] [*-f* _<from>_] [*-t* _<to>_] [*-c* _<cc>_] [*-d* _<start[,end]>_] [_<terms>_...]
	Searches the current folder for messages matching the given set of
	conditions.

	*:filter* restricts the displayed messages to only the search results.

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

	*-H*: Search in the headers of the messages, for a specific header
		Syntax: _Header: Value_
		If the space between _:_ and the value is omitted, aerc will not
		search for the Header.

	*-b*: Search in the body of the messages

	*-a*: Search in the entire text of the messages

	*-f* _<from>_: Search for messages from _<from>_

	*-t* _<to>_: Search for messages to _<to>_

	*-c* _<cc>_: Search for messages cc'ed to _<cc>_

	*-d* _<start[..end]>_:
		Search for messages within a particular date range defined as
		\[start, end) where the dates are in the YYYY-MM-DD format.

		Relative dates can be used to specify a date range. Spaces and
		underscores are allowed to improve readability:

		*today*, *yesterday*

		*(this|last) (year|month|week)*

		*Weekdays*, *Monthnames*
			Can also be abbreviate, so Monday..Tuesday can written
			as Mon..Tue and February..March as Feb..Mar.

		_<N>_ *(y[ear]|m[onth]|w[eek]|d[ay])*
			_<N>_ is a positive integer that represents the number
			of the time units in the past. Multiple relative terms
			can will be accumulated. The units can also be
			abbreviated by a single letter such that yesterday would
			correspond to _1d_ (equivalent to _1 day_ or _1_day_)
			and _8 days ago_ would be either _1w1d_ or _8d_.

# NOTMUCH

*:filter* _query_...
*:search* _query_...
	You can use the full notmuch query language as described in
	*notmuch-search-terms*(7).

	The query will only apply on top of the active folder query.

	Example, jump to next unread:

		:search tag:unread

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
