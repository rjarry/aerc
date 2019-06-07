#!/usr/bin/env python3
from colorama import Fore, Style
import sys
import re

ansi_escape = re.compile(r'\x1B\[[0-?]*[ -/]*[@-~]')
stat_re = re.compile(r'(| \d+ )(\+*)(\-*)')
lines_re = re.compile(r'@@ (-\d+,\d+ \+\d+,\d+) @@')

sys.stdin.reconfigure(encoding='utf-8', errors='ignore')
patch = sys.stdin.read().replace("\r\n", "\n")
patch = ansi_escape.sub('', patch)

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
                line = lines_re.sub(f"{Fore.CYAN}@@ \\1 @@{Style.RESET_ALL}",
                        line)
                print(line)
            else:
                print(f"{Style.BRIGHT}{line}{Style.RESET_ALL}")
    else:
        if line.startswith(" ") and "|" in line and ("+" in line or "-" in line):
            line = stat_re.sub(
                    f'\\1{Fore.GREEN}\\2{Fore.RED}\\3{Style.RESET_ALL}',
                    line)
        print(line)
