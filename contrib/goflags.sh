#!/bin/sh

set -e

tags=

if ${CC:-cc} -x c - -o/dev/null -lnotmuch 2>/dev/null; then
	tags="$tags,notmuch"
fi <<EOF
#include <notmuch.h>

#if !LIBNOTMUCH_CHECK_VERSION(5, 6, 0)
#error "aerc requires libnotmuch.so.5.6 or later"
#endif

void main(void) {
	notmuch_status_to_string(NOTMUCH_STATUS_SUCCESS);
}
EOF

if [ -n "$tags" ]; then
	printf -- '-tags=%s\n' "${tags#,}"
fi
