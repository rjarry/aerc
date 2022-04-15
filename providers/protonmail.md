---
title: "aerc-wiki: Providers/ProtonMail"
---

# ProtonMail

Using ProtonMail with aerc is not trivial, as you will likely
experience issues with the `protonmail-bridge` certificate, stored
in `$HOME/.config/protonmail/bridge/`. The workaround is to add this
certificate to the root trust database. Please note that this is tested
on Arch Linux, and might not be valid on other distros. Run this command:

```bash
sudo trust anchor --store ~/.config/protonmail/bridge/cert.pem
```

For outgoing mail to work, you might have to change the SMTP
configuration in `protonmail-bridge` and set security to SSL instead of
STARTTLS.

Example account configuration, using the default ports of
`protonmail-bridge`:

```ini
[Protonmail]
source   = imap+insecure://youraccount%40protonmail.com:yourprotonmailbridgepassword@127.0.0.1:1143
outgoing = smtps+plain://youraccount%40protonmail.com:yourprotonmailbridgepassword@127.0.0.1:1025
default  = INBOX
from     = Your Name <youraccount@protonmail.com>
copy-to  = Sent
```

The first time you run aerc with this configuration you can expect a
very long wait before anything shows up in your inbox. This will be
considerably faster on subsequent launches.
