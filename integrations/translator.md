---
title: "aerc-wiki: Integrations/translator"
---

# Translate Shell

Translation using [Translate Shell](https://www.soimort.org/translate-shell/)
can easily be integrated by adding the following to your binds.conf:

```ini
[view]
tr = :pipe trans -show-original n -b -no-autocorrect<Enter>
```

This will automatically attempt to detect the language of the received message
and translate it.

