---
title: "aerc-wiki: Providers/Microsoft\ Personal"
---

# Microsoft Email

If you have a personal Outlook or legacy Hotmail account, you won't be able to
authenticate with aerc normally using OAuth without some special configuration.
In this article, we will create a [Microsoft Entra][1] App, then give that
app the required scopes and permissions, then authenticate to that app with
[oama][2].

# Microsoft Entra

In this section, we will create a Microsoft Entra App with the required scopes
and permissions.

### Azure Account

First, create a [free](https://azure.microsoft.com/free/?WT.mc_id=A261C142F)
Azure account affiliated with your personal Outlook account. You will have to
enter credit card details, but your account will not be charged.

### Tenant

A tenant is a collection of user identities, apps, and groups that are managed
by Microsoft Entra ID. It's a secure boundary that controls access to an
organization's resources. Once your Azure account is created, a default tenant
called **Default Directory** will be created in your Entra Portal.

You can check your Tenants by going to the top-right of the [Microsoft Entra
Admin Center](https://entra.microsoft.com/#home) and seeing the name of the
tenant right below your username.

### Register an Application

Now, we will [register an app][3] with the Microsoft Identity platform

1. Sign in to the **Microsoft Entra admin center** as at least a *Cloud
   Application Administrator*.
2. If you have access to multiple tenants, use the Settings icon in the top
   menu to switch to the tenant in which you want to register the application
   from the Directories + subscriptions menu.
3. Browse to **Identity > Applications > App registrations** and select **New
   registration**.
4. Enter a display Name for your application -- call it something like
   `Aerc{YourUsername}`
5. Under **Who can use this application or access this API?**, select
   **Accounts in any organizational directory (Any Microsoft Entra ID tenant -
   Multitenant) Sand personal Microsoft accounts (e.g. Skype, Xbox)** -- this
   will allow your application to use the `common` tenant when
   [authenticating with Oama](#Using Oama) later.

### Giving the App Scopes

We must give our App the required scope permissions to access user's email
through IMAP and SMTP.

In the left menu, go to **App Registrations > All Applications > Your App**.
Then, under the **Api Permissions** menu...

1. Select **Add a Permission > Microsoft Graph API** and select the following
   scopes:

   - `email`
   - `offline_access`
   - `User.Read`
   - `Mail.ReadWrite`
   - `Mail.Send`
   - `IMAP.AccessAsUser.All`
   - `SMTP.Send`

2. Click **Grant admin consent for Default Directory** to grant consent for the
   scopes being added to the app.

![Giving the App Scopes Microsoft](providers/microsoft-scopes.png)

### Platform Configuration

Now, we will register our app as a Web application to be able to use it as an
authentication endpoint later.

At the left App Menu, go to **Authentication > Platform Configurations > Add
a platform** and select the `Web` platform. This is important because it will
enable us to pass in a client secret to our app's endpoint.

#### Redirect URIs

Add the following Redirect URIs to the Web platform:

- https://login.microsoftonline.com/common/oauth2/nativeclient
- http://localhost/

### Setting Up a Client Secret

We need a Client Secret in order to use OAuth. Click on **Certificates and
Secrets > Client Secrets > New Client Secret** and follow the directives to
create a new client secret.

Record the value of your client secret and keep it safe, because it won't be
shown to you later. Your client secret will also expire on the date that you
set it to, so you will have to repeat this process at that time.

Finally, go to **Overview** and record the value of your client ID.

# Using Oama

We are now ready to authenticate to our newly-created App using [Oama][2].

Download Oama from the latest Github releases or from the `oama-bin` package
on the AUR.

### Config File

Run `oama --help` for the first time and it will create a default config file
for you in `~/.config/oama/config.yaml`. Open this  config and add a section
under **services**, using the client id and secret from earlier:

```yaml
services:
  microsoft:
    # ...
  outlook:
    auth_endpoint: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
    token_endpoint: "https://login.microsoftonline.com/common/oauth2/v2.0/token"
    auth_http_method: GET
    token_params_mode: RequestBodyForm
    auth_scope: https://outlook.office.com/IMAP.AccessAsUser.All
      https://outlook.office.com/SMTP.Send
      offline_access
    client_id: CLIENTID
    client_secret: CLIENTSECRET
    tenant: common
    prompt: select_account
```

### Authentication

Run `oama authorize outlook yourname@email.com` and go to the `http://localhost
:portwhatever` as prompted, authenticate with your personal Outlook account,
and authorize your Entra app to access your account.

A token will be stored in your keyring that lets you authenticate via XOauth2.

# Back to Aerc

We are almost ready to connect with Aerc. First, we must enable IMAP access
in Outlook if you have not done so already.

### [Enabling IMAP in Outlook][4]

Go to [outlook.com](outlook.com) and access your personal Inbox. Then

1. Select **Settings > Mail > Forwarding and IMAP**.
2. Under POP and IMAP, toggle the slider for **Let devices and apps use IMAP**
3. Select Save

### Aerc Config
Add the following in your `aerc/accounts.conf`:

```
[AccountName]
source        = imaps+xoauth2://username%40email.com@outlook.office365.com:993
outgoing      = smtp+xoauth2://username%40email.com@outlook.office365.com:587
from          = Your Name <username@email.com>
cache-headers = true
source-cred-cmd   = oama access username@email.com
outgoing-cred-cmd = oama access username@email.com
```

And start aerc.

[1]: https://www.microsoft.com/en-us/security/business/microsoft-entra
[2]: https://github.com/pdobsan/oama
[3]: https://learn.microsoft.com/en-us/entra/identity-platform/quickstart-register-app?tabs=certificate
[4]: https://support.microsoft.com/en-us/office/pop-imap-and-smtp-settings-for-outlook-com-d088b986-291d-42b8-9564-9c414e2aa040
