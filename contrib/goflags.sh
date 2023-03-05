#!/bin/sh

set -e

tags=

if ${CC:-cc} -x c - -o/dev/null -lnotmuch; then
	tags="$tags,notmuch"
fi <<EOF
#include <notmuch.h>

void main(void) {
	notmuch_status_to_string(NOTMUCH_STATUS_SUCCESS);
}
EOF

if [ -n "$tags" ]; then
	printf -- '-tags=%s\n' "${tags#,}"
fi
