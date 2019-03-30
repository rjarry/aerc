#!/usr/bin/env python3
from colorama import Fore, Style
import sys
import re

patch = sys.stdin.read().replace("\r\n", "\n")
stat_re = re.compile(r'(\+*)(\-*)')
lines_re = re.compile(r'@@ (-\d+,\d+ \+\d+,\d+) @@')

hit_diff = False
for line in patch.split("\n"):
    if line.startswith("diff "):
        hit_diff = True
        print(f"{Style.BRIGHT}{line}{Style.RESET_ALL}")
        continue
    if hit_diff:
        if line.startswith("-"):
            print(f"{Fore.RED}{line}{Style.RESET_ALL}")
        elif line.startswith("+"):
            print(f"{Fore.GREEN}{line}{Style.RESET_ALL}")
        elif line.startswith(" "):
            print(line)
        else:
            if line.startswith("@@"):
                line = lines_re.sub(f"{Fore.BLUE}@@ \\1 @@{Style.RESET_ALL}",
                        line)
                print(line)
            else:
                print(f"{Style.BRIGHT}{line}{Style.RESET_ALL}")
    else:
        if line.startswith(" ") and "|" in line and ("+" in line or "-" in line):
            line = stat_re.sub(
                    f'{Fore.GREEN}\\1{Fore.RED}\\2{Style.RESET_ALL}',
                    line)
        print(line)
