# Drite CLI

Command line client for the Drite Studio customer API.

## Setup

```powershell
bun run src/index.ts auth login --token <dr_live_token>
```

The token is stored outside this repository in your user config directory. You can also skip login and use `DRITE_API_KEY`.

```powershell
$env:DRITE_API_KEY="<dr_live_token>"
```

Default API base URL is `https://dritestudio.co.th`.

```powershell
bun run src/index.ts config set-url https://dritestudio.co.th
```

## Examples

Open the interactive menu:

```powershell
bun run src/index.ts
```

```powershell
bun run src/index.ts me
bun run src/index.ts vps list
bun run src/index.ts vps stats <vps_id>
bun run src/index.ts vps start <vps_id>
bun run src/index.ts vps rename <vps_id> --name "Production VPS"
bun run src/index.ts hosting plans
bun run src/index.ts hosting list
bun run src/index.ts billing transactions --page 1 --limit 20
bun run src/index.ts ticket list --status open
bun run src/index.ts webhook list
```

For full API coverage, use `raw`.

```powershell
bun run src/index.ts raw GET /api/auth/me
bun run src/index.ts raw POST /api/auth/vps/<vps_id>/renew --json-file .\renew.json
bun run src/index.ts raw GET /api/auth/hosting/check-domain --query domain=example.com
```

On PowerShell, prefer `--json-file` or `--json @path\to\body.json` for raw JSON payloads because inline quote escaping is easy to get wrong.

## Build

```powershell
bun run build
```
