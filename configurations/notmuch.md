---
title: "aerc-wiki: Configurations/Notmuch"
---

# Emulating copy-to for the notmuch backend

Currently, the notmuch backend does not support the `copy-to` setting in
`accounts.conf`.

One way to work around this is to leverage `notmuch insert`. It works by
inserting the email being sent in the notmuch database and the maildir backing
it, which can then be synchronized to the remote IMAP server using your
favorite IMAP synchronization software.

The following script illustrates how this can be done. Note that it assumes your
setup uses a directory structure within your main notmuch maildir which consists
of:

- `$account/sent` for the Sent Emails folder
- `$account` is also the account name in [msmtp](https://marlam.de/msmtp/)

```shell
#!/bin/sh
# XXX: This does not handle encryption

# ensure the script ends whenever a command exits with non-zero status
set -e

EMAIL=`mktemp --suffix=.eml /tmp/XXXXXX`
clean_up() {
    rm -f $EMAIL
}

# The account to be used is given as the first argument of this script
account=$1
shift

# ensure clean_up() is called when we exit abnormally
trap 'clean_up' 0 1 2 3 15

# <stdin> of script gets the email, we save temporarily for using it twice
cat >$EMAIL

# First try to send the email, as it can cause more problems (i.e., connection)
# `set -e` prevents the mail from entering the database in case this fails.
# msmtp could be called with args from aerc, but --read-recipients already does the job
msmtp --account=$account --read-recipients --read-envelope-from <$EMAIL

# assumes all maildir accounts are configured with a 'sent' directory
# also make sure to tag it correctly
notmuch insert --folder=$account/sent -inbox -unread +sent <$EMAIL
```

If you call this script `aerc-notmuch-send`, the following can be set in
`accounts.conf` to ensure your emails are copied to your sent folder:

```ini
[myaccount]
from = My Name <my@email>
source = notmuch://YOUR_MAILDIR_PATH/
outgoing = /path/to/aerc-notmuch-send myaccount
```
