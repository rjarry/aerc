---
title: "aerc-wiki: Configurations/Colors"
---

# Custom plain text filter colours

To configure the
[colorize](https://git.sr.ht/~rjarry/aerc/tree/master/filters/colorize.c)
filter to use your preferred colour scheme, you can copy it into your home
directory and edit it as you wish.

You can then call the edited filter by setting the following in your
`aerc.conf`.

```ini
text/plain=awk -f ~/.config/aerc/filters/custom-colorize
```

- [solarized by Shaleen Jain](https://lists.sr.ht/~rjarry/aerc-devel/patches/30119#%3C20220310045758.228592-1-shaleen@jain.sh%3E+filters/colorize)
