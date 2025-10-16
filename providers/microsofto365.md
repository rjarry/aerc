---
title: "aerc-wiki: Providers/Microsoft (Office365)"
---

# Microsoft Email

Assuming IMAP access is enabled in the server, setting up aerc for Microsoft is
straightforward: the `accounts.conf` file needs to be modified post setup as
the outgoing emails will fail otherwise.

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

### Oama setup

We can use [oama][15] to authenticate with XOauth2.

Download Oama from the latest Github releases or from the `oama-bin` package
on the AUR.

Run `oama --help` for the first time and it will create a default config file
for you in `~/.config/oama/config.yaml`. Open this  config and edit the
"microsoft" section under "services" with the following fields:

```yaml
services:
  microsoft:
    client_id: 08162f7c-0fd2-4200-a84a-f25a4db0b584 # notsecret
    client_secret: 'TxRBilcHdC6WGBee]fs?QR:SJ8nI[g82' # notsecret
    auth_scope: https://outlook.office.com/IMAP.AccessAsUser.All
      https://outlook.office.com/SMTP.Send
      offline_access
    tenant: common
    prompt: select_account
```

#### Impersonating Thunderbird

Yes, the client id and secret above are exposed. This is because they are
**Thunderbird**'s default client id and secret. We will be using them to
impersonate Thunderbird as the authenticating application.

> [!IMPORTANT]
> If Thunderbird decides to rotate their client secret, we are SOL!

Now, run `oama authorize microsoft yourname@email.com` and go to the
`http://localhost:portwhatever` as prompted, authenticate with your Office365
account, and allow "Thunderbird" to access your email.

A token will be stored in your keyring that lets you authenticate via XOauth2.

### Aerc config

With oama authorized, setup your account in your aerc/accounts.config like so:

```ini
[OrganizationName]
source            = imaps+xoauth2://username%40email@outlook.office365.com?
outgoing          = smtp+xoauth2://username%40email@outlook.office365.com:587
from              = "Your Name" <username@email>
cache-headers     = true
source-cred-cmd   = oama access username@email
outgoing-cred-cmd = oama access username@email
```

and start aerc.

### Maildir setup

You can also use [mbsync][7] to sync your Office365 mailbox with a maildir.
First, you'll need to install the Cyrus SASL OAuth2 plugin as described on
[Stack Exchange][8]:

```bash
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

## Office365 with IMAP disabled

If your Office365 email provider has decided that IMAP is a thing of the past,
then you won't be allowed to use aerc, or that's what your provider will assume.
In that situation, you need to leverage the [Davmail][12] gateway.

With Davmail installed and running, you can access MS Exchange servers in their
different flavours, and you'll get a local IMAP server in return. Use that
server as your mail server inside aerc, and you're set. Of course, the server
being in the same machine as the client, you don't need any security:

```ini
source             = imap+insecure://you@email.com@localhost:1143
outgoing           = smtp+insecure://you@email.com@localhost:1025
smtp-starttls      = no
```

Given that, thanks to davmail, access to IMAP is still possible despite your
sysadmins concerns, you can also leverage mbsync to get a local Maildir copy of
your emails (and, in turn, enable notmuch on that copy). Your `mbsyncrc` account
definition might look like:

```
IMAPAccount o365-davmail
  Host localhost
  Port 1143
  User you@email.com
  Pass ""
  SSLType None
  AuthMech LOGIN
```

Finally, if your sysadmins are even stricter, they might even straightaway
forbid the use of different applications to access mail. If you find yourself
in that situation, you need to instruct Davmail to mask itself as the very fine
Outlook client, as explained [elsewhere][13]. In that case, some reports
indicate that you need to use Davmail's `O365Manual` login type. When using
`O365Manual` davmail will provide you with a link where you can authorize your
account using the usual procedure you would use to log in. The authorization
will end by opening a link with your access token in it (if the page doesn't
open, look under developer tools -> console in your browser). Since the link
and the access token are not tied to the computer where you are running
`davmail` if you are unable to authorize on your current computer (e.g. you are
running davmail in a headless environment), you can either copy the link to
a different machine and copy the token back. Alternatively, [carbonyl][14] runs
chromium in a terminal, complete with the necessary javascript capabilities to
access the authorization page on a headless machine.

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
[11]: https://github.com/gaoDean/oauthRefreshToken
[12]: https://davmail.sourceforge.net/
[13]: https://github.com/mguessan/davmail/issues/321#issuecomment-1867072418
[14]: https://github.com/fathyb/carbonyl
[15]: https://github.com/pdobsan/oama
