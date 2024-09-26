---
title: "aerc-wiki: Writing HTML mail"
---

Although we believe that [email should be written as plain
text](https://useplaintext.email/) and `aerc` is optimized for dealing with
regular `text/plain` messages, but the things happen in life when one’s boss or
some other evil interference requires one to write an HTML message. And `aerc`
can help you even in such circumstances (and, of course, it can help you
perfectly well with [reading HTML mail](htmlquote) as well).

In the `aerc.conf` configuration file, section `[multipart-converters]` you
have to add new value for the generated format you need, so for example:

```
text/html=rst2html5 --embed-stylesheet --no-doc-title
```

For writing `text/html` messages with
[reStructuredText](https://docutils.sourceforge.io/rst.html).

Then, when reviewing a message before its sending, one can run the command

```
:multipart text/html
```

to generate `multipart/alternative` message, where one part will be `text/html`
(and the other the original `text/plain` content).

Of course, it is the best to add a binding to the `[compose:review]` section of
the `binds.conf` configuration file for speed access to the functionality:

```
H = :multipart text/html<Enter>
```
