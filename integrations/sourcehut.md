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
