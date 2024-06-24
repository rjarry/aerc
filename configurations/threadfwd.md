---
title: "aerc-wiki: Configurations/Induce threading of forwarded messages"
---

If you want that the messages that you forward appear threaded to the original
message being forwarded, you can add the following header to your forwarding
template (for instance, `forward_as_body`):
```
References: {{ if (not (eq (.OriginalHeader "References") ""))}}{{.OriginalHeader "References"}} {{end}}{{.OriginalHeader "Message-Id"}}
```

Please note that the [RFC](https://www.rfc-editor.org/rfc/rfc5322#section-3.6.4)
only recommends adding this header to replies (although some clients seem to
add it to forwarded messages too). Also note that most likely the receiver won't
have any of the mails referenced in these headers.
