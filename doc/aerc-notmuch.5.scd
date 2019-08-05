aerc-notmuch(5)

# NAME

aerc-notmuch - notmuch configuration for *aerc*(1)

# SYNOPSIS

aerc supports using the notmuch email system as a backend, for fast indexing
and searching.

For this to be enabled, aerc needs to be build with notmuch support.++
Refer to the installation instructions for details.

# CONFIGURATION

Notmuch accounts currently are not supported with the :new-account command and
must be added manually.

In accounts.conf (see *aerc-config*(5)), the following notmuch-specific
options are available:

*source*
	notmuch://path

	The *source* indicates the path to the directory containing your notmuch
	database (usually a .notmuch folder).

	The path portion of the URL following _notmuch://_ must be either an absolute
	path prefixed by */* or a path relative to your home directory prefixed with
	*~*. For example:

		source = notmuch:///home/me/mail

		source = notmuch://~/mail

*query-map*
	Path to a file containing a mapping from display name to notmuch query in
	the form of NAME=QUERY.++
Multiple entries can be specified, one per line.

	e.g. inbox=tag:inbox and not tag:archived

	Default: none

*exclude-tags*
	Comma separated list of tags which will be excluded from query results,
	unless explicitly mentioned in the query.++
This can for example be useful if you use an archive or spam tag.

	Default: none

# SEE ALSO

*aerc*(1) *aerc-config*(5) *aerc-smtp*(5) *aerc-maildir*(5)

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
