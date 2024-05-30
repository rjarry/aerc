AERC-SEARCH(1)

# NAME

aerc-search - search and filter patterns and options for *aerc*(1)

# SYNTAX

This syntax is common to all backends.

*:filter* [*-ruba*] [*-x* _<flag>_] [*-X* _<flag>_] [*-H* _<header>:[<value>]_] [*-f* _<from>_] [*-t* _<to>_] [*-c* _<cc>_] [*-d* _<start[..end]>_] [_<terms>_...]++
*:search* [*-ruba*] [*-x* _<flag>_] [*-X* _<flag>_] [*-H* _<header>:[<value>]_] [*-f* _<from>_] [*-t* _<to>_] [*-c* _<cc>_] [*-d* _<start[..end]>_] [_<terms>_...]
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
			_Draft_
				Draft messages

	*-H* _<header>:[<value>]_:
		Search in the headers of the messages for a specific _<header>_ that matches _<value>_,
		_<value>_ can be omitted to only search for a the _<header>_.
		If either the _<header>_ or the _<value>_ contain a space then the whole argument needs
		to be escaped with quotes, note: spaces around _<value>_ are trimmed.

	*-b*: Search in the body of the messages

	*-a*: Search in the entire text of the messages

	*-f* _<from>_: Search for messages from _<from>_

	*-t* _<to>_: Search for messages to _<to>_

	*-c* _<cc>_: Search for messages cc'ed to _<cc>_

	*-d* _<start[..end]>_:
		Search for messages within a particular date range defined as
		\[start, end).

		Spaces and underscores are allowed in relative dates to improve
		readability.

		_YYYY-MM-DD_

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

For notmuch, it is possible to avoid using the above flags and only rely on
notmuch search syntax.

*:filter* _query_...++
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
