---
title: "aerc-wiki: Integrations/combine-address-books"
---

# Combine Address Books

In come cases you may want to use multiple address book sources for
autocompletion within aerc. Such as your local contacts and notmuch.

## A Shell Script

The simplest way to achieve this is to just use a shell script to wrap your
address book providers. This can be accomplished like so:

```bash
#! /bin/sh

khard email -a addressbook --parsable --remove-first-line "$1"
notmuch address "$1"
```

This takes the first argument supplied by aerc and returns queries from both of
them. Their order in the script also determines the order they will be
returned. So, if you want your `khard` contacts to come back first then it
should be above `notmuch` and so on.

## Using addr-book-combine

[addr-book-combine](https://jasoncarloscox.com/creations/addr-book-combine/)
is a utility just for doing this. It has many more robust options that the
above setup doesn't provide such as de-duplication of results.
