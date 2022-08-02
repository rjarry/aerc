---
title: "aerc-wiki: Integrations/SourceHut"
---

# Sourcehut Lists

This config entry allows setting
[headers](https://man.sr.ht/lists.sr.ht/#email-controls) to easily set the
status on patches.

```ini
[compose::review]
<C-r> = :choose \
    -o a approved "header X-Sourcehut-Patchset-Update APPROVED" \
    -o R Rejected "header X-Sourcehut-Patchset-Update REJECTED" \
    -o r needs-revision "header X-Sourcehut-Patchset-Update NEEDS_REVISION" \
    -o s superseded "header X-Sourcehut-Patchset-Update SUPERSEDED" \
    -o A Applied "header X-Sourcehut-Patchset-Update APPLIED" \
    <Enter>
```

To set a status hit Ctrl+R(eview) before sending the message.

# Thank-you Messages

> This section is under a CC-BY-SA License [Thanks to Drew
> DeVault](https://drewdevault.com/2022/07/25/Code-review-with-aerc.html)

Using this template thank-you messages to contributors can be pre-populated
with relevant information about the last push.

```
X-Sourcehut-Patchset-Update: APPLIED

Thanks!

{{exec `branch="$(git branch --show-current)"; { git remote get-url --push origin; git reflog -2 "origin/$branch" --pretty=format:%h | xargs printf '%s\n' | tac; } | xargs printf "To %s\n   %s..%s  $branch -> $branch"` ""}}
```

This template can be used by executing: `:reply -a -T[template-name]`
