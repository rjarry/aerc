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

do_test() {
	prefix="$1"
	tool_bin="$2"
	tool="$3"
	vec="$4"
	expected="$5"
	tmp=$(mktemp)
	dtmp=$(mktemp)
	status=0
	$prefix $tool_bin < $vec > $tmp || status=$?
	if [ $status -eq 0 ] && diff -u "$expected" "$tmp" > "$dtmp" 2>&1; then
		echo "ok      $tool < $vec > $tmp"
	else
		cat -vte "$dtmp" | while IFS= read -r line; do
			case "$line" in
			-*) printf '\e[31m%s\033[0m\n' "$line" ;;
			+*) printf '\e[32m%s\033[0m\n' "$line" ;;
			@*) printf '\e[36m%s\033[0m\n' "$line" ;;
			*) printf '%s\n' "$line" ;;
			esac
		done
		echo "error   $tool < $vec > $tmp [status=$status]"
		fail=1
	fi
	rm -f -- "$tmp" "$dtmp"
}

for vec in $here/vectors/*.in; do
	expected=${vec%%.in}.expected
	tool=$(basename $vec | sed 's/-.*//')
	tool_bin=$here/../$tool
	prefix="$FILTERS_TEST_PREFIX $FILTERS_TEST_BIN_PREFIX"
	# execute source directly (and omit $...BIN_PREFIX) for interpreted filters
	if ! [ -f "$tool_bin" ]; then
		tool_bin=$here/$tool
		prefix="$FILTERS_TEST_PREFIX"
	fi
	do_test "$prefix" "$tool_bin" "$tool" "$vec" "$expected"

	case $tool in # additional test runs
	calendar) # Awk
		if awk -W posix -- '' >/dev/null 2>&1; then
			# test POSIX-compatibility
			do_test "$prefix" "awk -W posix -f $tool_bin" \
				"$tool (posix)" "$vec" "$expected"
		else # "-W posix" is not supported and not ignored, skip test
			echo "?       $tool < $vec > $tmp [no '-W posix' support]"
		fi
		;;
	esac
done

exit $fail
