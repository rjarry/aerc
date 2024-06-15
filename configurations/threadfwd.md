---
title: "aerc-wiki: Configurations/Induce threading of forwarded messages"
---

If you want that the messages that you forward appear threaded to the original
messages being forwarded, you can add these headers to your forwarding template
(for instance, `forward_as_body`):

```
In-Reply-To: {{.OriginalHeader "Message-Id"}}
References: {{ if (not (eq (.OriginalHeader "References") ""))}}{{.OriginalHeader "References"}} {{end}}{{.OriginalHeader "Message-Id"}}
```

Please note that, according to the RFC, `In-Reply-To` header should be used only
for replies (although it seems a common practice to use it too for forwards).
Also note that most likely the receiver won't have any of the mails referenced
in these headers.
