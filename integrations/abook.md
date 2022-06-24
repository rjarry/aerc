---
title: "aerc-wiki: Integrations/abook"
---

# abook

To use abook with aerc, you can simply add the following line to your
`aerc.conf`.

```ini
address-book-cmd = abook --mutt-query "%s"
```

In some releases abook prints an empty line at the beginning. In this case, you
may want to remove it like this:


```ini
address-book-cmd = sh -c 'abook --mutt-query "%s" | tail -n +2'
```
