---
title: "aerc-wiki: Providers/Microsoft"
---

# Microsoft Email

Setting up aerc for Microsoft is straight forward but the `accounts.conf` file
needs to be modified post setup as the outgoing emails will fail otherwise.

The main thing that needs to be changed is the outgoing credentials need to
be set to `smtp+login`. The below settings have been made very generic as
Microsoft runs many subdomains such as Hotmail, Live, Outlook, and MSN.

```ini
[Microsoft]
source        = imaps://youraccount%40provider@imapprovider:993
outgoing      = smtp+login://youraccount%40provider@smtpprovider:587
default       = INBOX
smtp-starttls = yes
from          = Your Name <youraccount@microsoftaccount>
copy-to       = Sent
```

The website to check settings is located here [POP, IMAP, and SMTP Settings][1].

[1]: https://support.microsoft.com/en-us/office/pop-imap-and-smtp-settings-8361e398-8af4-4e97-b147-6c6c4ac95353
