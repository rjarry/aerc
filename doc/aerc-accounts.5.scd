AERC-ACCOUNTS(5)

# NAME

aerc-accounts - account configuration file format for *aerc*(1)

# SYNOPSIS

The _accounts.conf_ file is  used for configuring each mail account used for
aerc. It is expected to be in your XDG config home plus _aerc_, which defaults
to _~/.config/aerc/accounts.conf_. This file must be kept secret, as it may
include your account credentials. An alternate file can be specified via the
_--accounts-conf_ command line argument, see *aerc*(1).

If _accounts.conf_ does not exist, the *:new-account* configuration wizard will
be executed automatically on first startup.

This file is written in the ini format where each *[section]* is the name of an
account you want to configure, and the keys & values in that section specify
details of that account's configuration. In addition to the options documented
here, specific transports for incoming and outgoing emails may have additional
configuration parameters, documented on their respective man pages.

# CONFIGURATION

Note that many of these configuration options are written for you, such as
*source* and *outgoing*, when you run the account configuration wizard
(*:new-account*).

*archive* = _<folder>_
	Specifies a folder to use as the destination of the *:archive* command.

	Default: _Archive_

*check-mail* = _<duration>_
	Specifies an interval to check for new mail. Mail will be checked at
	startup, and every interval. IMAP accounts will check for mail in all
	unselected folders, and the selected folder will continue to receive
	PUSH mail notifications. Maildir/Notmuch folders must use
	*check-mail-cmd* in conjunction with this option. See *aerc-maildir*(5)
	and *aerc-notmuch*(5) for more information.

	Setting this option to _0_ will disable *check-mail*

	Example:
		*check-mail* = _5m_

	Default: _0_

*copy-to* = _<folder>_
	Specifies a folder to copy sent mails to, usually _Sent_.

*default* = _<folder>_
	Specifies the default folder to open in the message list when aerc
	configures this account.

	Default: _INBOX_

*folders* = _<folder1,folder2,folder3...>_
	Specifies the comma separated list of folders to display in the sidebar.
	Names prefixed with _~_ are interpreted as regular expressions.

	By default, all folders are displayed.

*folders-exclude* = _<folder1,folder2,folder3...>_
	Specifies the comma separated list of folders to exclude from the sidebar.
	Names prefixed with _~_ are interpreted as regular expressions.
	Note that this overrides anything from *folders*.

	By default, no folders are excluded.

*enable-folders-sort* = _true_|_false_
	If _true_, folders are sorted, first by specified folders (see *folders-sort*),
	then alphabetically.

	Default: _true_

*folders-sort* = _<folder1,folder2,folder3...>_
	Specifies a comma separated list of folders to be shown at the top of the
	list in the provided order. Remaining folders will be sorted alphabetically.

*folder-map* = _<file>_
	The folder map contains a one-to-one mapping of server folders to displayed
	folder names. The *folder-map* file expects a
	_<display-folder-name>_=_<server-folder-name>_[\*]
	mapping per line (similar key=value syntax as for the *query-map* in notmuch).
	The mappings are applied as they appear in the *folder-map*.
	Supported backends: imap, maildir.

	Note that other account options such as *archive*, *default*, *copy-to*,
	*postpone*, *folders*, *folders-exclude*, *folders-sort* need to be
	adjusted if one of those folders is affected by a folder mapping.

	To apply the mapping to subfolders or folders with a similar prefix,
	append '\*' to the server folder name.

	Examples:

	Remap a single folder:
	```
	Spam = [Gmail]/Spam
	```

	Remap the folder and all of its subfolders:
	```
	G = [Gmail]\*
	```

	Remove a prefix for all subfolders:
	```
	* = [Gmail]/\*
	```

	Remap all subfolders and avoid a folder collision:
	```
	Archive/existing = Archive\*
	Archive = OldArchive\*
	```

*from* = _<address>_
	The default value to use for the From header in new emails. This should be
	an RFC 5322-compatible string, such as _Your Name <you@example.org>_.

*aliases* = _<address1,address2,address3...>_
	All aliases of the current account. These will be used to fill in the From:
	field. Make sure that your email server accepts this value, or for example
	use *aerc-sendmail*(5) in combination with *msmtp*(1) and
	*--read-envelope-from*.

	An alias can also use fnmatch-style wildcards in the address portion. These
	wildcards can be useful for catch-all addresses. For example, the alias
	_"Your Name" <\*@you.com>_ would ensure that when replying to emails addressed
	to _hi@you.com_ and _contact@you.com_, the From: field is set to
	_hi@you.com_ and _contact@you.com_, respectively. The name from the alias,
	not from the matching address, is used.

*headers* = _<header1,header2,header3...>_
	Specifies the comma separated list of headers to fetch with the message.

	By default, all headers are fetched. If any headers are specified in this
	list, aerc will append it to the following list of required headers:

	- date
	- subject
	- from
	- sender
	- reply-to
	- to
	- cc
	- bcc
	- in-reply-to
	- message-id
	- references

*headers-exclude* = _<header1,header2,header3...>_
	Specifies the comma separated list of headers to exclude from fetching.
	Note that this overrides anything from *headers*.

	By default, no headers are excluded.

*outgoing* = _<uri>_
	Specifies the transport for sending outgoing emails on this account. It
	should be a connection string, and the specific meaning of each component
	varies depending on the protocol in use. See each protocol's man page for
	more details:

	- *aerc-sendmail*(5)
	- *aerc-smtp*(5)

*outgoing-cred-cmd* = _<command>_
	Specifies an optional command that is run to get the outgoing account's
	password. See each protocol's man page for more details.

*outgoing-cred-cmd-cache* = _true_|_false_
	By default, the credentials returned by the command will be cached until
	aerc is shut down. If set to _false_, *outgoing-cred-cmd* will be executed
	every time an email is to be sent.

	Default: _true_

*pgp-auto-sign* = _true_|_false_
	If _true_, all outgoing emails from this account will be signed (if a signing
	key is available).

	Default: _false_

*pgp-attach-key* = _true_|_false_
	If _true_, attach the public signing key to signed outgoing emails.

	Default: _false_

*pgp-self-encrypt* = _true_|_false_
	If _true_, any outgoing encrypted email will be also encrypted for the sender
	or the key specified in *pgp-key-id*.

	Default: _false_

*pgp-error-level* = _none_|_warn_|_error_
	The level of error to display when opportunistic encryption cannot be
	performed. See *pgp-opportunistic-encryption*.

	Default: _warn_

*pgp-key-id* = _<key-id>_
	Specify the key id to use when signing a message. Can be either short or
	long key id. If unset, aerc will look up the key by email.

*pgp-opportunistic-encrypt* = _true_|_false_
	If _true_, any outgoing email from this account will be encrypted when all
	recipients (including Cc and Bcc field) have a public key available in
	the keyring. The level of error to display when a message can't be
	encrypted can be configured with *pgp-error-level*.

	Default: _false_

*postpone* = _<folder>_
	Specifies the folder to save postponed messages to.

	Default: _Drafts_

*send-as-utc* = _true_|_false_
	Converts the timestamp of the Date header to UTC.

	Default: _false_

*send-with-hostname* = _true_|_false_
	Uses the local hostname in outgoing Message-Id headers instead of your
	email address domain name.

	Default: _false_

*source* = _<uri>_
	Specifies the source for reading incoming emails on this account. This key
	is required for all accounts. It should be a connection string, and the
	specific meaning of each component varies depending on the protocol in use.
	See each protocol's man page for more details:

	- *aerc-imap*(5)
	- *aerc-jmap*(5)
	- *aerc-maildir*(5)
	- *aerc-notmuch*(5)

*source-cred-cmd* = _<command>_
	Specifies an optional command that is run to get the source account's
	password. See each protocol's man page for more details.

*signature-file* = _<path>_
	Specifies the file to read in order to obtain the signature to be added
	to emails sent from this account.

	Please note that by convention the Usenet signature style of two dashes,
	followed by a space ("-- ") should be placed at the top of the signature
	to separate content and signature. Aerc will add that delimiter if it is
	not already present.

*signature-cmd* = _<command>_
	Specifies the command to execute with _sh -c_ in order to obtain the
	signature to be added to emails sent from this account. If the command
	fails then *signature-file* is used instead.

*trusted-authres* = _<host1,host2,host3...>_
	Comma-separated list of trustworthy hostnames from which the
	Authentication Results header will be displayed. Entries can be regular
	expressions. If you want to trust any host (e.g. for debugging),
	use the wildcard _\*_.

*subject-re-pattern* = _<regexp>_
	When replying to a message, this is the regular expression that will
	be used to match the prefix of the original message's subject that has
	to be removed, to create the subject line of the new message.
	Typically, this will be used to avoid a repetition of the Re:
	prefix in the subject header. The default will match known
	translations for the common Re:.

	Default: _(?i)^((AW|RE|SV|VS|ODP|R): ?)+_

# SEE ALSO

*aerc*(1) *aerc-config*(5) *aerc-imap*(5) *aerc-jmap*(5) *aerc-maildir*(5)
*aerc-notmuch*(5) *aerc-sendmail*(5) *aerc-smtp*(5)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
