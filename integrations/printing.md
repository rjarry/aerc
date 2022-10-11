---
title: "aerc-wiki: Integrations/printing"
---

# printing

You can use [email2pdf](https://github.com/andrewferrier/email2pdf) to print
emails, and the `choose` command to select printers.

First, install `email2pdf`. For example, on Arch:

```sh
yay -S email2pdf
```

Because `email2pdf` can't handle piped data, we need a little script to manage
some temporary files. Save this script somewhere in your path; for this
tutorial, we'll call it `emailprint`:

```sh
#!/bin/sh
# Print a piped email
# If an argument is provided, it is the printer name.
# The input is an email piped over stdin.

INPF="/tmp/mailprint_$$.txt"
OPDF="/tmp/mailprint_$$.pdf"

clean_temps() {
	rm -f "$INPF" "$OPDF"
}
trap 'clean_temps' 0 1 2 3 15
cat > $INPF

PRINTER=""
[[ -n $1 ]] && PRINTER="-d $1"

email2pdf -i "$INPF" -o "$OPDF"

if [[ $1 == "-" ]]; then
	mv "$OPDF" "$HOME"
	printf "Done; file is %s/%s\n" "$HOME" "$OPDF"
else
	lp $PRINTER "$OPDF"
fi
```

This script takes an argument (the printer name), or a dash (`-`).  The `-`,
simply moves the intermediate PDF to your home directory, allowing you to print
emails to PDF files. The script cleans up temporary files, and if you print to
file, tells you where the PDF is. In aerc, this message will appear when you're
done printing and will be dismissed when you press `<Enter>`.

Make sure the script is executable (e.g., `chmod +x emailprint`).

Finally, configure hotkeys in aerc to print an email, for example, in your
`$HOME/.config/aerc/binds.conf`:

```ini
[view]
p = :choose \
        -o m mx "pipe -m emailprint mx" \
        -o e epson "pipe -m emailprint epson" \
        -o f file "pipe -m emailprint -" \
        <Enter>
```

Make sure to replace the printer names with your printer(s). The pattern for
each printer is:

```ini
  -o <choicekey> <choicename> "pipe -m <printscript> <printername"
```

If you have more printers, add more of the `-o` lines.
