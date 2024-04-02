#!/bin/sh

set -xe

list="${1:-https://lists.sr.ht/~rjarry/aerc-devel}"
url="${2:-https://bot.diabeteman.com/sourcehut/}"

hut lists webhook create "$list" --stdin -e patchset_received -u "$url" <<EOF
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
					}
					... on Mailbox {
						name
						address
					}
				}
			}
		}
	}
}
EOF

hut lists webhook create "$list" --stdin -e email_received -u "$url" <<EOF
query {
	webhook {
		uuid
		event
		date
		... on EmailEvent {
			email {
				id
				subject
				patchset_update: header(want: "X-Sourcehut-Patchset-Update")
				references: header(want: "References")
				list {
					name
					owner {
						... on User {
							canonicalName
						}
					}
				}
			}
		}
	}
}
EOF
