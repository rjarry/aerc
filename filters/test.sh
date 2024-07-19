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
	expected=${vec%%.in}.expected
	tool=$(basename $vec | sed 's/-.*//')
	tool_bin=$here/../$tool
	prefix="$FILTERS_TEST_PREFIX $FILTERS_TEST_BIN_PREFIX"
	# execute source directly (and omit $...BIN_PREFIX) for interpreted filters
	[ -f $tool_bin ] || { tool_bin=$here/$tool; prefix="$FILTERS_TEST_PREFIX"; }
	tmp=$(mktemp)
	status=0
	$prefix $tool_bin < $vec > $tmp || status=$?
	if [ $status -eq 0 ] && diff -u "$expected" "$tmp"; then
		echo "ok      $tool < $vec > $tmp"
	else
		echo "error   $tool < $vec > $tmp [status=$status]"
		fail=1
	fi
	rm -f -- "$tmp"
done

exit $fail
