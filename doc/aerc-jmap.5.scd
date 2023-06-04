AERC-JMAP(5)

# NAME

aerc-jmap - JMAP configuration for *aerc*(1)

# SYNOPSIS

aerc implements the JMAP protocol as specified by RFCs 8620 and 8621.

# CONFIGURATION

JMAP accounts currently are not supported with the *:new-account* command and
must be added manually.

In _accounts.conf_ (see *aerc-accounts*(5)), the following JMAP-specific options
are available:

*source* = _<scheme>_://[_<username>_][_:<password>@_]_<hostname>_[_:<port>_]/_<path>_
	Remember that all fields must be URL encoded. The _@_ symbol, when URL
	encoded, is _%40_.

	_<hostname>_[_:<port>_]/_<path>_ is the HTTPS JMAP session resource as
	specified in RFC 8620 section 2 without the leading _https://_ scheme.

	Possible values of _<scheme>_ are:

	_jmap_
		JMAP over HTTPS using Basic authentication.

	_jmap+oauthbearer_
		JMAP over HTTPS using OAUTHBEARER authentication

		The username is ignored any may be left empty. If specifying the
		password, make sure to prefix it with _:_ to make it explicit
		that the username is empty. Or set the username to any random
		value. E.g.:

		```
		source = jmap+oauthbearer://:s3cr3t@example.com/jmap/session
		source = jmap+oauthbearer://me:s3cr3t@example.com/jmap/session
		```

	Your source credentials must have the _urn:ietf:params:jmap:mail_
	capability.

*source-cred-cmd* = _<command>_
	Specifies the command to run to get the password for the JMAP account.
	This command will be run using _sh -c command_. If a password is
	specified in the *source* option, the password will take precedence over
	this command.

	Example:
		source-cred-cmd = pass hostname/username

*outgoing* = _jmap://_
	The JMAP connection can also be used to send emails. No need to repeat
	the URL nor any credentials. Just the URL scheme will be enough.

	Your source credentials must have the _urn:ietf:params:jmap:submission_
	capability.

*cache-state* = _true_|_false_
	Cache all email state (mailboxes, email headers, mailbox contents, email
	flags, etc.) on disk in a levelDB database located in folder
	_~/.cache/aerc/<account>/state_.

	The cached data should remain small, in the order of a few megabytes,
	even for very large email stores. Aerc will make its best to purge
	deleted/outdated information. It is safe to delete that folder when aerc
	is not running and it will be recreated from scratch on next startup.

	Default: _false_

*cache-blobs* = _true_|_false_
	Cache all downloaded email bodies and attachments on disk as individual
	files in _~/.cache/aerc/<account>/blobs/<xx>/<blob_id>_ (where _<xx>_ is
	a subfolder named after the last two characters of _<blob_id>_).

	Aerc will not purge the cached blobs automatically. Even when their
	related emails are destroyed permanently from the server. If required,
	you may want to run some periodic cleanup based on file creation date in
	a crontab, e.g.:

		@daily find ~/.cache/aerc/foo/blobs -type f -mtime +30 -delete

	Default: _false_

*use-labels* = _true_|_false_
	If set to _true_, mailboxes with the _archive_ role (usually _Archive_)
	will be hidden from the directory list and replaced by an *all-mail*
	virtual folder. The name of that folder can be configured via the
	*all-mail* setting.

	*:archive flat* may still be used to effectively "tag" messages with the
	hidden _Archive_ mailbox so that they appear in the *all-mail* virtual
	folder. When the *all-mail* virtual folder is selected, *:archive flat*
	should not be used and will have no effect. The messages will be grayed
	out but will never be refreshed until aerc is restarted.

	Also, this enables support for the *:modify-labels* (alias *:tag*)
	command.

	Default: _false_

*all-mail* = _<name>_
	Name of the virtual folder that replaces the role=_archive_ mailbox when
	*use-labels* = _true_.

	Default: _All mail_

*server-ping* = _<duration>_
	Interval the server should ping the client at when monitoring for email
	changes. The server may choose to ignore this value. By default, no ping
	will be requested from the server.

	See https://pkg.go.dev/time#ParseDuration.

# NOTES

JMAP messages can be seen as "labels" or "tags". Every message must belong to
one or more mailboxes (folders in aerc). Each mailbox has a "role" as described
in _https://www.iana.org/assignments/imap-mailbox-name-attributes/_.

When deleting messages that belong only to the selected mailbox, aerc will
attempt to "move" these messages to a mailbox with the _trash_ role. If it
cannot find such mailbox or if the selected mailbox is the _trash_ mailbox, it
will effectively destroy the messages from the server.

*:delete* removes messages from the selected mailbox and effectively does the
same thing than *:tag -<selected_folder>*.

*:cp <foo>* is an alias for *:tag <foo>* or *:tag +<foo>*.

*:mv <foo>* is a compound of *:delete* and *:mv* and can be seen as an alias of
*:tag -<selected_folder> +<foo>*.

*:archive flat* is an alias for *:tag -<selected_folder> +<archive>*.

# SEE ALSO

*aerc*(1) *aerc-accounts*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
