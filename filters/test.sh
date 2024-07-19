#!/bin/sh

set -e

here=$(dirname $0)
fail=0
style=$(mktemp)
trap "rm -f $style" EXIT
cat >$style <<EOF
# stuff

url.fg = red

[viewer]
*.normal=true
*.default=true
url.underline = true # cxwlkj
header.bold=    true  # comment
signature.dim=true
diff_meta.bold    =true
diff_chunk.dim=		true
invalid . xxx = lkjfdslkjfdsqqqqqlkjdsq
diff_add.fg= #00ff00 # comment
# comment
diff_del.fg=     1		# comment2
quote_*.fg     =6
quote_*.dim=true
quote_1.dim=false

[user]
foo = bar
EOF
export AERC_STYLESET=$style
export AERC_OSC8_URLS=1

for vec in $here/vectors/*.in; do
	tool=$(basename $vec | sed 's/-.*//')
	expected=${vec%%.in}.expected
	tmp=$(mktemp)
	status=0
	$FILTERS_TEST_PREFIX $here/../$tool -f $vec > $tmp || status=$?
	if [ $status -eq 0 ] && diff -u "$expected" "$tmp"; then
		echo "ok      $tool < $vec > $tmp"
	else
		echo "error   $tool < $vec > $tmp [status=$status]"
		fail=1
	fi
	rm -f -- "$tmp"
done

exit $fail
