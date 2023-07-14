#!/bin/sh

set -xe

hut lists webhook create "https://lists.sr.ht/~rjarry/aerc-devel" \
	--stdin -e patchset_received \
	-u https://bot.diabeteman.com/sourcehut/ <<EOF
query {
	webhook {
		uuid
		event
		date
		... on PatchsetEvent {
			patchset {
				id
				subject
				version
				prefix
				list {
					name
					owner {
						... on User {
							canonicalName
						}
					}
				}
				submitter {
					... on User {
						canonicalName
						username
						email
					}
				}
			}
		}
	}
}
EOF
