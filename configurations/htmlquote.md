---
title: "aerc-wiki: Configurations/HTML quoting in replies"
---

Sometimes the email you get only has `text/html`. If you quote reply this, the
default [quoted_reply template][quoted_reply] will dump the entire html with
all the markup into the quote. You can update this template to behave
differently if replying to html. For example, add the html filter shipped with
aerc to construct a more sensible text to quote:

```
{{ if eq .OriginalMIMEType "text/html" -}}
	{{- trimSignature (exec `~/.local/softwarefromsource/aerc/filters/html` .OriginalText) | quote -}}
{{- else -}}
	{{- trimSignature .OriginalText | quote -}}
{{- end}}
```

This can be expanded to also properly quote html sent as text/plain and to make
the filtered content more readable by dropping empty lines, links, and leading
whitespace.

```
{{ if or
	(eq .OriginalMIMEType "text/html")
	(contains (toLower .OriginalText) "<html")
}}
	{{- $text := exec `/usr/lib/aerc/filters/html` .OriginalText | replace `\r` `` -}}
	{{- range split "\n" $text -}}
		{{- if eq . "References:" }}{{break}}{{end}}
		{{- if or
			(eq (len .) 0)
			(match `^\[.+\]\s*$` .)
		}}{{continue}}{{end}}
		{{- printf "%s\n" . | replace `^[\s]+` "" | quote}}
	{{- end -}}
{{- else }}
	{{- trimSignature .OriginalText | quote -}}
{{- end -}}

{{.Signature -}}
```

Note, that since this filter comes with a third-party dependency (w3m), this
will not be added to the default template shipped with aerc.

[quoted_reply]: https://git.sr.ht/~rjarry/aerc/tree/master/item/templates/quoted_reply
