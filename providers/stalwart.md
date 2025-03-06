---
title: "aerc-wiki: Providers/Stalwart"
---

# Stalwart

[Stalwart](https://stalw.art/) is an all-in-one open-source mail server. It supports JMAP and IMAP. It is strongly recommended that you configure it with JMAP. It will be much faster and reliable than IMAP.

## Account Configuration

You'll eventually want to modify `$XDG_CONFIG/aerc/accounts.conf` to contain something like the following:


```
[me@mydoman.tld]
source      = jmap://username:password@mydomain.tld/.well-known/jmap
outgoing    = jmap://
default     = INBOX
from        = John Smith <me@mydomain.tld>
use-labels  = true
cache-state = true
cache-blobs = false
```

The critical item is that the `source` should be set to the JMAP `.well-known` URL. This will allow aerc to authenticate and query Stalwart for the various endpoints for getting and sending mail.
