#!/bin/sh

set -e

tmp=$(mktemp -d)
trap "rm -rf $tmp" EXIT

cat > $tmp/src.go <<EOF
package main

// #cgo LDFLAGS: -lnotmuch
// #include <notmuch.h>
import "C"

func main() {
	C.notmuch_status_to_string(C.NOTMUCH_STATUS_SUCCESS)
}
EOF

${GO:-go} build -o $tmp/out $tmp/src.go
