---
title: "aerc-wiki: Integrations/emailbook"
---

# emailbook

To use emailbook with aerc, you can add the following line to your
`aerc.conf`, `[compose]` section:

```ini
address-book-cmd = emailbook /path/to/emailbook.txt --search "%s"
```

Set a keybinding in `binds.conf`, `[view]` section, for adding all e-mail
addresses of the current e-mail to your emailbook:

```ini
aa = :pipe -m emailbook /path/to/emailbook.txt --parse --all<Enter>
```

Project page:
[emailbook: A minimalistic address book for e-mails only (mainly for aerc)](https://sr.ht/~maxgyver83/emailbook/)
