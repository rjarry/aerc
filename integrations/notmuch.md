---
title: "aerc-wiki: notmuch"
---

# Protonmail with mbsync and notmuch

The following details a working setup with the notmuch extension,
allowing you to keep a local copy of all your emails and work with it
offline, deciding for yourself when you want to sync with the ProtonMail
server. With small changes this can easily be converted to other email
providers as well by setting the appropriate hostname.

Example account configuration using the notmuch extension:

```ini
[Protonmail]
source    = notmuch://~/mail/
query-map = ~/.config/aerc/map.conf
outgoing = smtp+plain://youraccount%40protonmail.com:yourprotonmailbridgepassword27.0.0.1:1025
default  = INBOX
from     = Your Name <youraccount@protonmail.com>
copy-to  = Sent
smtp-starttls = yes
```
You will need a query-map file to populate the sidebar list with
pre-defined searches.

Example map.conf:

```ini
Inbox=tag:inbox and not tag:archived and not tag:deleted
```
# mbsync configuration

In order to use `notmuch` you can use `mbsync`
or `offlineimap` to synchronize your emails from the remote server to
your local machine. One possible setup for this is with the following
`~/.mbsyncrc`:

```ini
IMAPAccount protonmail
Host 127.0.0.1
Port 1143
User youraccount@protonmail.com
Pass yourUnH4ckablePassw0rd
SSLType NONE

IMAPStore pm-remote
Account protonmail

MaildirStore pm-local
Path ~/mail/
Inbox ~/mail/INBOX/

Channel pm-inbox
Far :pm-remote:
Near :pm-local:
Patterns "INBOX"
Create Both
Expunge Both
SyncState *

Channel pm-sent
Far :pm-remote:"Sent"
Near :pm-local:"sent"
Create Both
Expunge Both
SyncState *

Group protonmail
Channel pm-inbox
Channel pm-sent
```
More tips and tricks on using mbsync can be found in the [Arch
Wiki](https://wiki.archlinux.org/title/Isync).

# notmuch configuration

Next you need to configure notmuch to create a searchable database.
This is an example of `~/.notmuch-config`:

```ini
[database]

path=/home/username/mail

[user]
name=Your Name
primary_email=youraccount@protonmail.com

[new]
tags=unread;inbox;sent;
ignore=

[search]
exclude_tags=deleted;spam;
[maildir]
synchronize_flags=true

[crypto]
gpg_path=gpg
```

# Syncronizing with the remote server

The syncronization can be done manually by running this command:

```bash
mbsync -Va && notmuch new
```
This does not, however, delete mails you have tagged as deleted. For
this you need to run something like this:

```bash
notmuch search --format=text0 --output=files tag:deleted | xargs -0 --no-run-if-empty rm -v
```

Putting it together, you get this script `mail-sync.sh`:

```bash
#!/bin/sh

MBSYNC=$(pgrep mbsync)
NOTMUCH=$(pgrep notmuch)

if [ -n "$MBSYNC" -o -n "$NOTMUCH" ]; then
    echo "Already running one instance of mbsync or notmuch. Exiting..."
    exit 0
fi

echo "Deleting messages tagged as *deleted*"
notmuch search --format=text0 --output=files tag:deleted | xargs -0 --no-run-if-empty rm -v

mbsync -Va
notmuch new
```

Make sure the script is in your `$PATH `and is executable.

You may want to run this script with a `systemd` timer or a `cron` job,
or bind it to a keyboard shortcut instead in `~/.config/aerc/binds.conf`
instead:
```ini
[messages]
o = :exec mail-sync.sh<Enter>
```

Alternatively, you can use a utility like `goimapnotify` to run the
script whenever a new email has arrived. Using this configuration
`~/.config/imapnotify/protonmail.conf`:

```ini
{
  "host": "127.0.0.1",
  "port": 1143,
  "tls": false,
  "tlsOptions": {
    "rejectUnauthorized": false
  },
  "username": "kennethflak@protonmail.com",
  "password": "yourprotonmailbridgepassword",
  "onNewMail": "/home/user/bin/mail-sync.sh",
  "wait": 20,
  "boxes": [ "INBOX", "Sent" ]
}
```

Another approach can be found here:
[mbsyncwatch.py](https://git.sr.ht/~rjarry/dotfiles/tree/fdbea3ba273cd696b1ab001c6aeaba14a71320a9/item/bin/mbsyncwatch.py),
accomplishing the same thing as the `goimapnotify` `mail-sync.sh`
combination in a possibly more robust way. This does not, however, add
any `notmuch` post processing commands, so you would have to run this
manually, or add this to the script yourself.

# Deleting emails

You will not be able to delete emails with the `:delete-message`
command when using notmuch. The solution for this is to use
`:modify-labels +deleted` instead. This can be mapped to a key in
`~/.config/aerc/binds.conf` like this:

```ini
[messages]
md = :modify-labels +deleted<Enter>
```

# Notmuch as Address Book Provider

It is possible to use notmuch as your address book. This will make any
aerc source addresses from any email you have ever sent or received. To
make this work add this to `aerc.conf`:
```ini
address-book-cmd='notmuch address "%s"'
```
