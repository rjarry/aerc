---
title: "aerc-wiki: Integrations/aercbook"
---

# aercbook

To use aercbook with aerc, you can add the following line to your
`aerc.conf`, `[compose]` section:

```ini
address-book-cmd = aercbook /path/to/aercbook.txt "%s"
```

Set a keybinding in `binds.conf`, `[view]` section, for adding all e-mail
addresses of the current e-mail to your aercbook:

```ini
aa = :pipe -m aercbook /path/to/aercbook.txt --parse --add-all<Enter>
```

Project page:
[aercbook: Minimalistic address book for aerc](https://sr.ht/~renerocksai/aercbook/)
