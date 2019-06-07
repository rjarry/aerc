#!/usr/bin/env python3
from colorama import Fore, Style
import sys
import re

# TODO: Wrap text to terminal width?

# TODO: I guess this might vary from MUA to MUA. I've definitely seen localized
# versions in the wild
quote_prefix_re = re.compile(r"On .*, .* wrote:")
quote_re = re.compile(r">+")

sys.stdin.reconfigure(encoding='utf-8', errors='ignore')
mail = sys.stdin.read().replace("\r\n", "\n")

for line in mail.split("\n"):
    if quote_re.match(line) or quote_prefix_re.match(line):
        print(f"{Style.DIM}{Fore.CYAN}{line}{Style.RESET_ALL}")
    else:
        print(line)
