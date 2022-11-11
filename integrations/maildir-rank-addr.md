---
title: "aerc-wiki: Integrations/maildir-rank-addr"
---

# maildir-rank-addr

To use [maildir-rank-addr](https://github.com/ferdinandyb/maildir-rank-addr)
with aerc, you need to first run `maildir-rank-addr` at least once, but likely,
you want to run it periodically, like every 12 hours. You can either set up
a cronjob, or systemd timer.

Once you have the generated `addressbook.tsv` set up your favourite grep in
`aerc.conf` as your address book command, e.g.:

```ini
address-book-cmd=grep -i -m 100 %s /home/[myuser]/.cache/maildir-rank-addr/addressbook.tsv
```

Note, that aerc only displays the first 100 entries for the completion so no
point in giving it more.
