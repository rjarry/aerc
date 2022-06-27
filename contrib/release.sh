#!/bin/sh

set -e

echo "======= Determining next version..."
prev_tag=$(git describe --tags --abbrev=0)
next_tag=$(echo $prev_tag | awk -F. -v OFS=. '{$(NF-1) += 1; print}')
read -rp "next tag ($next_tag)? " n
if [ -n "$n" ]; then
	next_tag="$n"
fi

echo "======= Updating version in Makefile..."
sed -i Makefile -e "s/$prev_tag/$next_tag/g"
git add Makefile
git commit -sm "Release version $next_tag"

echo "======= Creating tag..."
git tag --edit --sign \
	-m "Release $next_tag highlights:" \
	-m "$(git log --format='- %s' $prev_tag..)" \
	-m "Thanks to all contributors!" \
	-m "~\$ git shortlog -sn $prev_tag..$next_tag
$(git shortlog -sn $prev_tag..)" \
	"$next_tag"

echo "======= Pushing to remote..."
git push origin master "$next_tag"

echo "======= Sending release email..."

email=$(mktemp aerc-release-XXXXXXXX.eml)
trap "rm -f -- $email" EXIT

cat >"$email" <<EOF
To: aerc-annouce <~rjarry/aerc-announce@lists.sr.ht>
Cc: aerc <~sircmpwn/aerc@lists.sr.ht>
Reply-To: aerc-devel <~rjarry/aerc-devel@lists.sr.ht>
Subject: aerc $next_tag
User-Agent: aerc/$next_tag
Message-ID: <$(date +%Y%m%d%H%M%S).$(base64 -w20 < /dev/urandom | head -n1)@$(hostname)>

Hi all,

I am glad to announce the release of aerc $next_tag.

https://git.sr.ht/~rjarry/aerc/refs/$next_tag

$(git tag -l --format='%(contents)' "$next_tag" | sed -n '/BEGIN PGP SIGNATURE/q;p')
EOF

$EDITOR "$email"

/usr/sbin/sendmail -t < "$email"
