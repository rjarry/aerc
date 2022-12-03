---
title: "aerc-wiki: Providers/Microsoft"
---

# Microsoft Email

Setting up aerc for Microsoft is straight forward but the `accounts.conf` file
needs to be modified post setup as the outgoing emails will fail otherwise.

The main thing that needs to be changed is the outgoing credentials need to
be set to `smtp+login`. The below settings have been made very generic as
Microsoft runs many subdomains such as Hotmail, Live, Outlook, and MSN.

```ini
[Microsoft]
source        = imaps://youraccount%40provider@imapprovider:993
outgoing      = smtp+login://youraccount%40provider@smtpprovider:587
default       = INBOX
smtp-starttls = yes
from          = Your Name <youraccount@microsoftaccount>
copy-to       = Sent
```

The website to check settings is located here [POP, IMAP, and SMTP Settings][1].

## Office365 with XOAUTH2

Office365 sometimes uses XOAUTH2, which is a bit of a pain to setup.
Instructions are provided below. This topic has been discussed [multiple][9]
[times][10] on the mailing list as well -- those threads may have additional
useful information.

The first step is to use a script such as [`mutt_oauth2.py`][2] or [oauth2ms][3]
to fetch a token. With `mutt_oauth2.py`, the basic steps are as follows:

1. Download the [script][2] and make it executable.
2. Modify the `microsoft` section of the `registrations` dictionary based on
   your Office365 setup. You'll likely need to modify the `tenant`, `client_id`,
   and `client_secret`, as well as the `*_endpoint` and `redirect_uri` fields,
   replacing `common` with the value used for `tenant`. There are some
   instructions provided by [oauth2ms][4] and [OfflineIMAP][5] that may help
   with finding these values.
3. Do an initial run of the script to obtain a token: `./mutt_oauth2.py
   /path/to/token --verbose --authorize`. You can choose where to store the
   token. Answer the questions, choosing `localhostauthcode` when asked, and
   follow the instructions to visit the authorization webpage. (See also
   [vanormondt.net][6].)

Once you've followed these steps, you should be able to print a token by running
`./mutt_oauth2.py /path/to/token`.

Finally, you can add the Office365 account to aerc's `accounts.conf`:

```ini
source            = imaps+xoauth2://you%40email.com@outlook.office365.com
source-cred-cmd   = /path/to/mutt_oauth2.py /path/to/token
outgoing          = smtp+xoauth2://you%40email.com@outlook.office365.com:587
outgoing-cred-cmd = /path/to/mutt_oauth2.py /path/to/token
smtp-starttls     = yes
```

### Maildir setup

You can also use [mbsync][7] to sync your Office365 mailbox with a maildir.
First, you'll need to install the Cyrus SASL OAuth2 plugin as described on [Stak
Exchange][8]:

```
git clone https://github.com/moriyoshi/cyrus-sasl-xoauth2.git

# Configure and make.
cd cyrus-sasl-xoauth2
./autogen.sh
./configure

# SASL2 libraries on Ubuntu are in /usr/lib/x86_64-linux-gnu/; modify the Makefile accordingly
sed -i 's%pkglibdir = ${CYRUS_SASL_PREFIX}/lib/sasl2%pkglibdir = ${CYRUS_SASL_PREFIX}/lib/x86_64-linux-gnu/sasl2%' Makefile

make
sudo make install

# Verify XOAUTH2 is known to SASL.
saslpluginviewer | grep XOAUTH2
```

Note that you may need to modify the `sed` command to ensure the libraries get
put in the correct place for your system, and `saslpluginviewer` may have a
different name on your system. For example, on Arch Linux the libraries need to
go in `/usr/lib64/sasl2/` and `saslpluginviewer` is just `pluginviewer`.

Once you have this plugin setup, you can use XOAUTH2 in your `.mbsyncrc`:

```
IMAPAccount you@email.com
Host outlook.office365.com
User you@email.com
AuthMechs XOAUTH2
PassCmd "/path/to/mutt_oauth2.py /path/to/token"
SSLType IMAPS
```

(That isn't the full config -- you'll need to also setup an `IMAPStore`,
`MaildirStore`, and `Channel`, but you can reference the mbsync docs for that.)

Then simply setup a Maildir account for aerc as described in aerc-maildir(5).

[1]: https://support.microsoft.com/en-us/office/pop-imap-and-smtp-settings-8361e398-8af4-4e97-b147-6c6c4ac95353
[2]: https://gitlab.com/muttmua/mutt/-/blob/master/contrib/mutt_oauth2.py
[3]: https://github.com/harishkrupo/oauth2ms
[4]: https://github.com/harishkrupo/oauth2ms/blob/main/steps.org
[5]: https://github.com/UvA-FNWI/M365-IMAP
[6]: https://www.vanormondt.net/~peter/blog/2021-03-16-mutt-office365-mfa.html
[7]: https://github.com/gburd/isync
[8]: https://unix.stackexchange.com/questions/625637/configuring-mbsync-with-authmech-xoauth2
[9]: https://lists.sr.ht/~rjarry/aerc-discuss/%3CCA%2BrC5JmSTNDTd%3DKB0h-NeXRExB2QpHCWCOXch4%2BA%3DCiTX0wFAw%40mail.gmail.com%3E
[10]: https://lists.sr.ht/~rjarry/aerc-discuss/%3CCNKU4TGF41CJ.3HIV0H45QQWU2%40manjaro%3E