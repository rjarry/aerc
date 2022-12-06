AERC-NOTMUCH(5)

# NAME

aerc-notmuch - notmuch configuration for *aerc*(1)

# SYNOPSIS

aerc supports using the notmuch email system as a backend, for fast indexing
and searching.

For this to be enabled, aerc needs to be build with notmuch support.
Refer to the installation instructions for details.

# CONFIGURATION

Notmuch accounts currently are not supported with the *:new-account* command and
must be added manually.

In _accounts.conf_ (see *aerc-accounts*(5)), the following notmuch-specific
options are available:

*check-mail-cmd* = _<command>_
	Command to run in conjunction with *check-mail* option.

	Example:
		check-mail-cmd = mbsync -a

*check-mail-timeout* = _<duration>_
	Timeout for the *check-mail-cmd*. The command will be stopped if it does
	not complete in this interval and an error will be displayed. Increase from
	the default if repeated errors occur

	Default: _10s_

*source* = notmuch://_<path>_
	The *source* indicates the path to the directory containing your notmuch
	database (usually a _.notmuch/_ folder).

	The path portion of the URL following _notmuch://_ must be either an absolute
	path prefixed by _/_ or a path relative to your home directory prefixed with
	_~_. For example:

		source = notmuch:///home/me/mail

		source = notmuch://~/mail

*query-map* = _<file>_
	Path to a file containing a mapping from display name to notmuch query
	in the form of *<NAME>*=_<QUERY>_.

	Multiple entries can be specified, one per line. Lines starting with _#_
	are ignored and serve as comments.

	e.g. inbox=tag:inbox and not tag:archived

*exclude-tags* = _<tag1,tag2,tag3...>_
	Comma separated list of tags which will be excluded from query results,
	unless explicitly mentioned in the query.

	This can for example be useful if you use an _archive_ or _spam_ tag.

*maildir-store* = _<path>_
	Path to the maildir store containing the message files backing the
	notmuch database. This is often the same as the notmuch database path.
	This is optional. If specified, it will be used by aerc to list
	available folders and enable commands such as *:delete* and *:archive*.

# USAGE

Notmuch shows slightly different behavior than for example imap. Some commands
are slightly different in semantics and mentioned below:

*cf* _<notmuch query>_
	The change folder command allows for arbitrary notmuch queries. Performing a
	*:cf* command will perform a new top-level notmuch query.

*filter* _<notmuch query>_
	The filter command for notmuch backends takes in arbitrary notmuch queries.
	It applies the query on the set of messages shown in the message list. This
	can be used to perform successive filters/queries. It is equivalent to
	performing a set of queries concatenated with "and".

# SEE ALSO

*aerc*(1) *aerc-accounts*(5) *aerc-smtp*(5) *aerc-maildir*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
