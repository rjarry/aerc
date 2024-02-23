---
title: "aerc-wiki: Configurations/Filters"
---

# Filter examples by MIME type

## `application/gzip`

- `tar -z --list`

## `application/pgp-keys`

- `gpg`

## `image/gif`

- `catimg -w "$(tput cols)" -`

## `image/jpeg`

- `catimg -w "$(tput cols)" -`

## `image/gif`

- `catimg -w "$(tput cols)" -`

## `text/html`

- `lynx -assume_charset=UTF-8 -display_charset=UTF-8 -localhost -stdin -dump | colorize`

## `text/markdown`

- `glow --style dark --width "$(tput cols)"`

	Note that without applying a `--style` argument, you may not get colored
	output.
