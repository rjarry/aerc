#!/bin/sh

set -e

dry_run=false
case "$1" in
-n|--dry-run)
	dry_run=true
	;;
esac

changelog() {
	title_prefix=$1
	width=$2
	first=true
	wrap=cat
	if [ -n "$width" ]; then
		wrap="./wrap -r -w$width"
	fi
	for kind in Added Fixed Changed Deprecated; do
		format="%(trailers:key=Changelog-$kind,unfold,valueonly)"
		if git log --format="$format" $prev_tag.. | grep -q .; then
			if [ "$first" = true ]; then
				first=false
			else
				echo
			fi
			echo "$title_prefix $kind"
			echo
			git log --reverse --format="$format" $prev_tag.. | \
				sed -E '/^$/d; s/[[:space:]]+/ /; s/^/- /' | \
				$wrap
		fi
	done
}

echo "======= Determining next version..."
prev_tag=$(git describe --tags --abbrev=0)
next_tag=$(echo $prev_tag | awk -F. -v OFS=. '{$(NF-1) += 1; print}')
read -rp "next tag ($next_tag)? " n
if [ -n "$n" ]; then
	next_tag="$n"
fi
tag_url="https://git.sr.ht/~rjarry/aerc/refs/$next_tag"

if [ "$dry_run" = false ]; then
	echo "======= Creating release commit..."
	sed -i GNUmakefile -e "s/$prev_tag/$next_tag/g"
	make wrap
	{
		echo
		echo "## [$next_tag]($tag_url) - $(date +%Y-%m-%d)"
		echo
		changelog "###" 80
	} > .changelog.md
	sed -i CHANGELOG.md -e '/^The format is based on/ r .changelog.md'
	${EDITOR:-vi} CHANGELOG.md
	rm -f .changelog.md
	git add GNUmakefile CHANGELOG.md
	git commit -vesm "Release version $next_tag"

	echo "======= Creating tag..."
	git -c core.commentchar='%' tag --edit --sign \
		-m "Release $next_tag highlights:" \
		-m "$(changelog '#' 72)" \
		-m "Thanks to all contributors!" \
		-m "~\$ contrib/git-stats.sh $prev_tag..$next_tag
$(contrib/git-stats.sh $prev_tag..)" \
		"$next_tag"

	echo "======= Pushing to remote..."
	git push origin master "$next_tag"
fi

echo "======= Sending release email..."

email=$(mktemp aerc-release-XXXXXXXX.eml)
trap "rm -f -- $email" EXIT

cat >"$email" <<EOF
From: $(git config user.name) <$(git config user.email)>
To: aerc-annouce <~rjarry/aerc-announce@lists.sr.ht>
Cc: aerc-devel <~rjarry/aerc-devel@lists.sr.ht>
Bcc: aerc <~sircmpwn/aerc@lists.sr.ht>,
	$(git config user.name) <$(git config user.email)>
Reply-To: aerc-devel <~rjarry/aerc-devel@lists.sr.ht>
Subject: aerc $next_tag
User-Agent: aerc/$next_tag
Message-ID: <$(date +%Y%m%d%H%M%S).$(base32 -w12 < /dev/urandom | head -n1)@$(hostname)>
Content-Transfer-Encoding: 8bit
Content-Type: text/plain; charset=UTF-8
MIME-Version: 1.0

Hi all,

I am glad to announce the release of aerc $next_tag.

$tag_url

Release highlights:

$(changelog '#')

# Changed dependencies for downstream packagers

$(contrib/depends-diff.py $prev_tag..)

Thanks to all contributors!

~\$ contrib/git-stats.sh $prev_tag..$next_tag

$(contrib/git-stats.sh $prev_tag..)
EOF

${EDITOR:-vi} "$email"

if [ "$dry_run" = false ]; then
	/usr/sbin/sendmail -t < "$email"
fi
