#!/bin/sh

set -e

here=$(dirname $0)
fail=0
export AERC_OSC8_URLS=1

for vec in $here/vectors/*.in; do
	tool=$(basename $vec | sed 's/-.*//')
	expected=${vec%%.in}.expected
	tmp=$(mktemp)
	if ! $FILTERS_TEST_PREFIX $here/../$tool -f $vec > $tmp; then
		fail=1
	fi
	if diff -u "$expected" "$tmp"; then
		echo "ok      $tool < $vec > $tmp"
	else
		echo "error   $tool < $vec > $tmp"
		fail=1
	fi
	rm -f -- "$tmp"
done

exit $fail
