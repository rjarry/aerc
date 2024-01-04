---
title: "aerc-wiki: Configurations/HTML quoting in replies"
---

Sometimes the email you get only has `text/html`. If you quote reply this, the
default [quoted_reply
template](https://git.sr.ht/~rjarry/aerc/tree/master/item/templates/quoted_reply)
will dump the entire html with all the markup into the quote. You can update
this template to behave differently if replying to html. For example, add the
html filter shipped with aerc to construct a more sensible text to quote:

```
{{ if eq .OriginalMIMEType "text/html" -}}
{{- trimSignature (exec `~/.local/softwarefromsource/aerc/filters/html` .OriginalText) | quote -}}
{{- else -}}
{{- trimSignature .OriginalText | quote -}}
{{- end}}
```

Note, that since this filter comes with a third-party dependency (w3m), this
will not be added to the default template shipped with aerc.
