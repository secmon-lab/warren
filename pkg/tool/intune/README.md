# Microsoft Intune Tool Configuration

## Overview

The Intune tool enables Warren to query Microsoft Intune managed device information via Microsoft Graph API. Security analysts can look up device compliance state, OS details, encryption status, and recent sign-in IP history from alert context (user email or device hostname).

This is a **read-only** tool — it does not modify any data in your Intune environment.

## Prerequisites

### 1. Register an Application in Azure AD

1. Sign in to the [Microsoft Entra admin center](https://entra.microsoft.com/)
2. Click **App registrations** in the left sidebar
3. Click **New registration**
4. Configure:
   - **Name**: A descriptive name (e.g., `warren-intune-readonly`)
   - **Supported account types**: "Single tenant only - {Your Organization}" (the default)
   - **Redirect URI**: Leave blank (not needed for client credentials flow)
5. Click **Register**
6. Note the **Application (client) ID** and **Directory (tenant) ID** from the Overview page

### 2. Create a Client Secret

1. In the app registration, go to **Certificates & secrets**
2. Click **New client secret**
3. Set a description (e.g., `warren`) and expiration period
4. Click **Add**
5. **Copy the "Value" column immediately** (not "Secret ID") — it cannot be retrieved later

### 3. Grant API Permissions

1. In the app registration, go to **API permissions**
2. Click **Add a permission > Microsoft Graph > Application permissions**
3. Add the following permissions:

| Permission | Purpose |
|---|---|
| `DeviceManagementManagedDevices.Read.All` | Read Intune managed device information |
| `AuditLog.Read.All` | Read sign-in logs for IP address history |

4. Click **Grant admin consent for [your organization]** (requires Global Administrator or Privileged Role Administrator)

> **Note:** `AuditLog.Read.All` is optional. If not granted, the tool still returns device information but without sign-in IP history.

## Deployment

### Environment Variables

```bash
export WARREN_INTUNE_TENANT_ID="your-azure-ad-tenant-id"
export WARREN_INTUNE_CLIENT_ID="your-application-client-id"
export WARREN_INTUNE_CLIENT_SECRET="your-client-secret-value"
# Optional: override Graph API base URL (default: https://graph.microsoft.com/v1.0)
# export WARREN_INTUNE_BASE_URL="https://graph.microsoft.com/v1.0"
```

### CLI Flags

```bash
warren serve \
  --intune-tenant-id="your-tenant-id" \
  --intune-client-id="your-client-id" \
  --intune-client-secret="your-client-secret"
```

### Cloud Run

```bash
gcloud run services update warren \
  --set-env-vars="WARREN_INTUNE_TENANT_ID=your-tenant-id" \
  --set-env-vars="WARREN_INTUNE_CLIENT_ID=your-client-id" \
  --set-env-vars="WARREN_INTUNE_CLIENT_SECRET=your-client-secret"
```

> **Security Note:** For production, use Secret Manager for the client secret:
> ```bash
> gcloud run services update warren \
>   --set-secrets="WARREN_INTUNE_CLIENT_SECRET=intune-secret:latest"
> ```

## Authentication

The tool uses [OAuth 2.0 Client Credentials Flow](https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-client-creds-grant-flow):

1. Sends `client_id` and `client_secret` to `POST https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/token`
2. Receives a bearer token (typically valid for 1 hour)
3. Caches the token and automatically refreshes before expiry (5-minute buffer)
4. On 401 responses, clears the cache and retries once

No manual token management is required.

## Available Tools

| Tool | Description |
|---|---|
| `intune_devices_by_user` | Search managed devices by user email / UPN |
| `intune_devices_by_hostname` | Search managed device by device hostname |

Both tools return:
- **Device details**: compliance state, OS, encryption, owner, model, serial number, Azure AD registration, management agent, MAC addresses, etc.
- **Sign-in IP history** (up to 50 recent entries): IP address, timestamp, client app, device detail

## Usage Examples

```
> Look up devices for user@example.com in Intune
> Check the compliance state of device LAPTOP-ABC123
> What devices does john.doe@company.com have?
```

## Troubleshooting

### Tool not appearing in available tools

Verify that all three required environment variables are set (`TENANT_ID`, `CLIENT_ID`, `CLIENT_SECRET`). The tool is skipped if any is empty.

### 401 Unauthorized errors

- Verify the Client ID and Client Secret are correct
- Check that the client secret has not expired in the Azure portal
- Ensure the Tenant ID matches your Azure AD directory

### 403 Forbidden errors

- Verify the app has `DeviceManagementManagedDevices.Read.All` permission
- Confirm admin consent has been granted
- For sign-in logs: verify `AuditLog.Read.All` permission (optional — device info still works without it)

### Empty results

- Verify the user UPN or device name exists in your Intune environment
- Check that devices are enrolled and syncing with Intune
- Confirm the app registration is in the same tenant as the managed devices
