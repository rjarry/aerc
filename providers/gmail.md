---
title: "aerc-wiki: Providers/Gmail"
---

# Gmail

Gmail servers support a relatively OK implementation of IMAP. The simplest way
to get started is to [enable 2FA and create an app password][app-password] for
aerc. Replace `$APP_PASSWORD` with your created password in the examples below.

[app-password]: https://support.google.com/mail/answer/185833

Then, follow the new account wizard and you should get something that looks
like this (replace `Gmail` with the account name of your choice):

```ini
[youraccount]
from     = Your Name <youraccount@gmail.com>
source   = imaps://youraccount%40gmail.com:$APP_PASSWORD@imap.gmail.com
outgoing = smtps://youraccount%40gmail.com:$APP_PASSWORD@smtp.gmail.com
```

It is recommended to enable some settings for a better experience:

```ini
default       = INBOX
folders-sort  = INBOX
postpone      = [Gmail]/Drafts
cache-headers = true

# Only enable copy-to for gmail if you use a different SMTP server.
# Gmail SMTP server will automatically tag sent messages properly.
#copy-to =

# To be able to use your google contacts. It only works for personal accounts, not enterprise.
carddav-source          = https://youraccount%40gmail.com@www.googleapis.com/carddav/v1/principals/youraccount@gmail.com/lists/default
carddav-source-cred-cmd = echo $APP_PASSWORD
address-book-cmd        = carddav-query -S youraccount %s
```
