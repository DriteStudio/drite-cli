# Drite Studio Customer API Docs

เอกสารนี้เป็นคู่มือหลักสำหรับใช้งาน Drite Studio Customer API และ Drite CLI
ผ่าน API key ของลูกค้า โดย endpoint ส่วนใหญ่ใช้สิทธิ์เดียวกับบัญชีเจ้าของ key
ดังนั้น key จะเห็นและจัดการได้เฉพาะ service ของบัญชีนั้นเท่านั้น

Repository: https://github.com/DriteStudio/drite-cli

## Base URL

Production:

```text
https://dritestudio.co.th
```

Authenticated API prefix:

```text
/api/auth
```

Public plan endpoints บางตัวอยู่ใต้:

```text
/api/un_auth
```

## Authentication

ทุก request ที่ต้อง login ใช้ header:

```http
Authorization: Bearer <dr_live_token>
Accept: application/json
Content-Type: application/json
```

ตัวอย่าง:

```bash
curl -X GET "https://dritestudio.co.th/api/auth/me" \
  -H "Authorization: Bearer dr_live_xxx" \
  -H "Accept: application/json"
```

ใช้ CLI login:

```powershell
bun run src/index.ts auth login --token <dr_live_token>
```

หรือใช้ environment variable:

```powershell
$env:DRITE_API_KEY="<dr_live_token>"
```

ตั้ง API base URL:

```powershell
bun run src/index.ts config set-url https://dritestudio.co.th
```

ตรวจสถานะ:

```powershell
bun run src/index.ts doctor
bun run src/index.ts auth status
```

## Output Format

ค่า default ของ CLI จะพิมพ์ JSON แบบอ่านง่าย:

```powershell
bun run src/index.ts vps list
```

ใช้ `--compact` ถ้าต้องการ JSON บรรทัดเดียวสำหรับ script:

```powershell
bun run src/index.ts vps list --compact
```

## Common Error Shape

API อาจตอบ error ได้หลายรูปแบบตาม endpoint แต่โดยทั่วไปจะมี `message`:

```json
{
  "message": "Unauthorized"
}
```

สำหรับบาง flow จะมี `code` เพิ่ม:

```json
{
  "message": "No available IPs",
  "code": "VPS_NO_AVAILABLE_IPS",
  "nextIpAvailableAt": "2026-05-05T12:00:00.000Z",
  "ipReleaseRequiresArchive": true
}
```

HTTP status ที่พบบ่อย:

| HTTP | ความหมาย |
| --- | --- |
| 200 | สำเร็จ |
| 201 | สร้างรายการสำเร็จ |
| 202 | รับงานเข้า queue แล้ว |
| 400 | payload หรือ parameter ไม่ถูกต้อง |
| 401 | token ไม่ถูกต้องหรือหมดอายุ |
| 403 | บัญชีไม่มีสิทธิ์ |
| 404 | ไม่พบ resource |
| 409 | state ชนกัน เช่น service กำลังทำงานอื่นอยู่ |
| 422 | validation ไม่ผ่าน |
| 429 | rate limit |
| 500 | server-side error |
| 503 | service หรือ infrastructure ยังไม่พร้อม |

## Query Parameters

ใช้ CLI `--query key=value` สำหรับ query ทั่วไป:

```powershell
bun run src/index.ts raw GET /api/auth/transactions --query page=1 --query limit=20
```

หลาย command มี flag ลัด เช่น:

```powershell
bun run src/index.ts billing transactions --page 1 --limit 20
bun run src/index.ts ticket list --status open
```

## Raw Requests

ใช้ `raw` เมื่อ endpoint ยังไม่มี command เฉพาะ:

```powershell
bun run src/index.ts raw GET /api/auth/me
bun run src/index.ts raw GET /api/auth/vps/<vps_id>
bun run src/index.ts raw POST /api/auth/vps/<vps_id>/renew --json-file .\renew.json
```

PowerShell แนะนำใช้ `--json-file` หรือ `--json @file.json` เพื่อลดปัญหา escape quote:

```powershell
bun run src/index.ts raw POST /api/auth/vps/<id>/rename --json-file .\rename.json
```

## Account API

### Get Profile

```http
GET /api/auth/me
```

CLI:

```powershell
bun run src/index.ts me
```

ใช้ตรวจ user, balance, role, API key status และข้อมูลบัญชีที่ endpoint ส่งกลับ

## VPS API

### List VPS

```http
GET /api/auth/vps
```

CLI:

```powershell
bun run src/index.ts vps list
```

Response:

```json
{
  "data": [
    {
      "id": "vps_id",
      "name": "production-01",
      "cpu": 2,
      "ram": 4,
      "disk": 80,
      "ip": "203.0.113.10",
      "duration": 30,
      "os": "Ubuntu",
      "version": "22.04 (TH)",
      "price": 300,
      "renew_at": "2026-06-01T00:00:00.000Z",
      "status": "online",
      "autoRenewal": true
    }
  ]
}
```

### VPS Plans

```http
GET /api/auth/vps/plans
GET /api/auth/vps/plans?templateId=<template_id>
```

CLI:

```powershell
bun run src/index.ts vps plans
bun run src/index.ts vps plans --template-id <template_id>
bun run src/index.ts vps plans --available-only
```

Response fields ที่สำคัญ:

```json
{
  "data": [
    {
      "id": "plan_id",
      "name": "VPS 2C 4G",
      "cpu": 2,
      "ram": 4,
      "disk": 80,
      "dailyPrice": 20,
      "weeklyPrice": 100,
      "monthlyPrice": 300,
      "yearlyPrice": 3000,
      "isOrderable": true,
      "availabilityStatus": "orderable",
      "availabilityReason": null,
      "nextIpAvailableAt": null
    }
  ]
}
```

Availability statuses:

| status | ความหมาย | ควรทำอย่างไร |
| --- | --- | --- |
| `orderable` | เช่าได้ | แสดง/เลือก plan ได้ |
| `no_capacity` | host ที่พร้อมใช้งานยังรองรับสเปคนี้ไม่ได้ | เลือก plan เล็กลง หรือรอ capacity เพิ่ม |
| `out_of_stock` | resource พอแต่ IP หมด | รอ IP ว่างหรือเลือก region/template อื่น |
| `not_public` | plan ไม่ public | ไม่ควรแสดงให้ลูกค้าเช่า |
| `disabled` | ถูกปิดใช้งาน | ไม่ควรแสดงให้ลูกค้าเช่า |

เมื่อ `out_of_stock` และมี `nextIpAvailableAt` ให้สื่อสารกับลูกค้าว่า IP จะว่างหลังเวลานั้นโดยประมาณ
เพราะ VPS ที่หมดอายุต้องถูก archive/convert เป็น template ก่อน IP จึงถูกปล่อยกลับ pool

CLI จะเติม field ช่วยอ่าน:

```json
{
  "availabilityBadge": "IP หมด",
  "availabilityMessage": "IP สำหรับเช่า VPS หมดชั่วคราว คาดว่าจะว่างหลัง ...",
  "availabilitySummary": {
    "total": 4,
    "orderable": 3,
    "blocked": 1
  }
}
```

### VPS Templates

```http
GET /api/auth/vps/templates
```

CLI:

```powershell
bun run src/index.ts vps templates
```

Response:

```json
{
  "data": [
    {
      "id": "template_id",
      "os": "Ubuntu",
      "version": "22.04 (TH)"
    }
  ]
}
```

ควรเลือก template ก่อนเรียก `/vps/plans?templateId=...` เพื่อรู้ว่า plan ไหนใช้กับ template/host นั้นได้จริง

### Available IP Count

```http
GET /api/auth/vps/available-ips/{hostId}
```

CLI:

```powershell
bun run src/index.ts vps available-ips <host_ip>
```

Response:

```json
{
  "data": {
    "available": 3,
    "nextIpAvailableAt": null
  }
}
```

### Create VPS

```http
POST /api/auth/vps
```

Body:

```json
{
  "name": "production-01",
  "templateId": "template_id",
  "planId": "plan_id",
  "durationType": "monthly",
  "password": "StrongPassw0rd!",
  "ip": "203.0.113.10",
  "networkRef": "OpaqueRef:..."
}
```

`ip` และ `networkRef` เป็น optional ปกติให้ระบบเลือกเอง

CLI:

```powershell
bun run src/index.ts vps create `
  --name production-01 `
  --template-id <template_id> `
  --plan-id <plan_id> `
  --duration monthly `
  --password "StrongPassw0rd!" `
  --wait
```

Response:

```json
{
  "data": {
    "id": "vps_id",
    "status": "working",
    "ip": "203.0.113.10"
  },
  "jobId": "vps_id",
  "message": "VPS provisioning started"
}
```

Create errors ที่ควรรู้:

| code/message | HTTP | ความหมาย |
| --- | --- | --- |
| `INSUFFICIENT_BALANCE` / `Insufficient balance` | 400 | ยอดเงินไม่พอ |
| `VPS_NO_CAPACITY` | 503 | ไม่มี host รองรับสเปค/OS นี้ตอนนี้ |
| `VPS_NO_AVAILABLE_IPS` | 400/503 | IP หมด |
| `VPS_TEMPLATE_UNAVAILABLE` | 409 | template ที่เลือกไม่อยู่บน Xen แล้ว |
| `IP_IN_USE` | 409 | IP ถูกใช้งานหรือ reserve อยู่ |
| `IP_BLACKLISTED` | 400 | IP ถูก blacklist |

### Job Status

```http
GET /api/auth/vps/job/{jobId}
```

CLI:

```powershell
bun run src/index.ts vps job <job_id>
```

หรือใช้ `--wait` กับ command ที่ queue งาน:

```powershell
bun run src/index.ts vps start <vps_id> --wait
bun run src/index.ts vps create --json-file .\create-vps.json --wait
```

### Get VPS Detail

```http
GET /api/auth/vps/{id}
```

CLI:

```powershell
bun run src/index.ts vps get <vps_id>
```

### Live Status

```http
GET /api/auth/vps/{id}/status
```

CLI:

```powershell
bun run src/index.ts vps status <vps_id>
bun run src/index.ts vps watch <vps_id>
```

### Stats

```http
GET /api/auth/vps/{id}/stats
```

CLI:

```powershell
bun run src/index.ts vps stats <vps_id>
```

### Activity

```http
GET /api/auth/vps/{id}/activity
```

CLI:

```powershell
bun run src/index.ts vps activity <vps_id>
```

### Control Actions

```http
POST /api/auth/vps/{id}/start
POST /api/auth/vps/{id}/stop
POST /api/auth/vps/{id}/reboot
POST /api/auth/vps/{id}/force-stop
POST /api/auth/vps/{id}/control
```

CLI:

```powershell
bun run src/index.ts vps start <vps_id> --wait
bun run src/index.ts vps stop <vps_id> --wait
bun run src/index.ts vps reboot <vps_id> --wait
bun run src/index.ts vps force-stop <vps_id> --wait
bun run src/index.ts vps control <vps_id> --action reboot --wait
```

`control` body:

```json
{
  "action": "start"
}
```

Allowed action: `start`, `stop`, `reboot`, `force-stop`

### Renew VPS

```http
POST /api/auth/vps/{id}/renew
```

Body:

```json
{
  "durationType": "monthly"
}
```

CLI:

```powershell
bun run src/index.ts vps renew <vps_id> --duration monthly
```

### Auto Renewal

```http
POST /api/auth/vps/{id}/auto-renewal
```

Body:

```json
{
  "enabled": true
}
```

CLI:

```powershell
bun run src/index.ts vps auto-renew <vps_id> --enabled true
```

### Upgrade Options

```http
GET /api/auth/vps/{id}/upgrade-options
```

CLI:

```powershell
bun run src/index.ts vps upgrade-options <vps_id>
```

Response มี `capacity.canUpgrade` เพื่อบอกว่า host เดิมรองรับ upgrade หรือไม่

### Upgrade VPS

```http
POST /api/auth/vps/{id}/upgrade
```

Body:

```json
{
  "planId": "new_plan_id"
}
```

CLI:

```powershell
bun run src/index.ts vps upgrade <vps_id> --plan-id <plan_id> --wait
```

### Rename VPS

```http
POST /api/auth/vps/{id}/rename
```

Body:

```json
{
  "name": "Production VPS"
}
```

CLI:

```powershell
bun run src/index.ts vps rename <vps_id> --name "Production VPS"
```

### Reinstall OS

```http
POST /api/auth/vps/{id}/reinstall
```

Body:

```json
{
  "templateId": "template_id",
  "password": "StrongPassw0rd!"
}
```

CLI:

```powershell
bun run src/index.ts vps reinstall <vps_id> --template-id <template_id> --password "StrongPassw0rd!" --wait
```

### Reset Password

```http
POST /api/auth/vps/{id}/reset-password
```

Body:

```json
{
  "password": "NewStrongPassw0rd!"
}
```

CLI:

```powershell
bun run src/index.ts vps reset-password <vps_id> --password "NewStrongPassw0rd!" --wait
```

### Network Reset

```http
POST /api/auth/vps/{id}/network-reset
```

CLI:

```powershell
bun run src/index.ts vps network-reset <vps_id> --wait
```

### Delete VPS

```http
DELETE /api/auth/vps/{id}
```

CLI:

```powershell
bun run src/index.ts vps delete <vps_id> --wait
```

### Failed VPS

```http
GET /api/auth/vps/failed
DELETE /api/auth/vps/failed/{id}
```

CLI:

```powershell
bun run src/index.ts vps failed
bun run src/index.ts vps ack-failed <failed_id>
```

## Hosting API

### Hosting Plans

```http
GET /api/un_auth/hosting/plans
```

CLI:

```powershell
bun run src/index.ts hosting plans
```

### List Hosting

```http
GET /api/auth/hosting/list
```

CLI:

```powershell
bun run src/index.ts hosting list
```

### Check Domain

```http
GET /api/auth/hosting/check-domain?domain=example.com
```

CLI:

```powershell
bun run src/index.ts hosting check-domain example.com
bun run src/index.ts hosting check-domain --domain example.com
```

### Deploy Hosting

```http
POST /api/auth/hosting/deploy
```

Body:

```json
{
  "planId": "plan_id",
  "duration": 30,
  "domain": "example.com",
  "password": "StrongPassw0rd!"
}
```

CLI:

```powershell
bun run src/index.ts hosting deploy `
  --plan-id <plan_id> `
  --duration monthly `
  --domain example.com `
  --password "StrongPassw0rd!"
```

หมายเหตุ: CLI ส่ง `durationType` สำหรับบาง endpoint และส่ง day number สำหรับ hosting deploy ตาม flow ที่ backend รองรับ

### Hosting Detail

```http
GET /api/auth/hosting/{id}
```

CLI:

```powershell
bun run src/index.ts hosting get <hosting_id>
```

### Hosting Stats

```http
GET /api/auth/hosting/{id}/stats
GET /api/auth/hosting/{id}/disk
GET /api/auth/hosting/{id}/traffic
GET /api/auth/hosting/{id}/activity
GET /api/auth/hosting/{id}/activation-status
```

CLI:

```powershell
bun run src/index.ts hosting stats <hosting_id>
bun run src/index.ts hosting disk <hosting_id>
bun run src/index.ts hosting traffic <hosting_id>
bun run src/index.ts hosting activity <hosting_id>
bun run src/index.ts hosting activation-status <hosting_id>
```

### Hosting Access

```http
POST /api/auth/hosting/{id}/access
```

CLI:

```powershell
bun run src/index.ts hosting access <hosting_id>
```

### Renew Hosting

```http
POST /api/auth/hosting/{id}/renew
```

Body:

```json
{
  "durationType": "monthly"
}
```

CLI:

```powershell
bun run src/index.ts hosting renew <hosting_id> --duration monthly
```

### Auto Renew Hosting

```http
POST /api/auth/hosting/{id}/autorenew
```

CLI:

```powershell
bun run src/index.ts hosting autorenew <hosting_id>
```

### Reset Hosting Password

```http
POST /api/auth/hosting/{id}/reset-password
```

Body:

```json
{
  "password": "NewStrongPassw0rd!"
}
```

CLI:

```powershell
bun run src/index.ts hosting reset-password <hosting_id> --password "NewStrongPassw0rd!"
```

### Delete Hosting

```http
DELETE /api/auth/hosting/{id}
```

CLI:

```powershell
bun run src/index.ts hosting delete <hosting_id>
```

## Public Plans API

### All Plans

```http
GET /api/un_auth/plans/all
```

CLI:

```powershell
bun run src/index.ts plans all
```

### Hosting Plans

```http
GET /api/un_auth/hosting/plans
```

CLI:

```powershell
bun run src/index.ts plans hosting
```

## Billing API

### Transactions

```http
GET /api/auth/transactions?page=1&limit=20
GET /api/auth/transactions/export?month=2026-05
```

CLI:

```powershell
bun run src/index.ts billing transactions --page 1 --limit 20
bun run src/index.ts billing export --month 2026-05
```

### Topup History and Status

```http
GET /api/auth/topup/history
GET /api/auth/topup/status/{referenceNo}
```

CLI:

```powershell
bun run src/index.ts billing topup-history
bun run src/index.ts billing topup-status <reference_no>
```

## Ticket API

### List Tickets

```http
GET /api/auth/ticket/list?page=1&limit=20&status=all
```

CLI:

```powershell
bun run src/index.ts ticket list --page 1 --limit 20 --status all
```

### Get Ticket

```http
GET /api/auth/ticket/{id}
```

CLI:

```powershell
bun run src/index.ts ticket get <ticket_id>
```

### Create Ticket

```http
POST /api/auth/ticket
```

Body:

```json
{
  "subject": "Need help with VPS",
  "category": "technical",
  "priority": "normal",
  "message": "Please check my VPS."
}
```

CLI:

```powershell
bun run src/index.ts ticket create `
  --subject "Need help with VPS" `
  --category technical `
  --priority normal `
  --message "Please check my VPS."
```

### Reply Ticket

```http
POST /api/auth/ticket/{id}/reply
```

Body:

```json
{
  "message": "More detail..."
}
```

CLI:

```powershell
bun run src/index.ts ticket reply <ticket_id> --message "More detail..."
```

### Close Ticket

```http
PUT /api/auth/ticket/{id}/close
```

CLI:

```powershell
bun run src/index.ts ticket close <ticket_id>
```

### Upload Attachment

```http
POST /api/auth/ticket/upload-url
POST /api/auth/ticket/upload
```

CLI:

```powershell
bun run src/index.ts ticket upload-url --filename screenshot.png --mime image/png
bun run src/index.ts ticket upload --file attachment=.\screenshot.png
```

## API Key Management

### API Key Status

```http
GET /api/auth/me
```

CLI:

```powershell
bun run src/index.ts api-key status
```

### Create API Key

```http
POST /api/auth/me/api-key
```

CLI:

```powershell
bun run src/index.ts api-key create
```

สำคัญ: full key จะแสดงครั้งเดียว ให้ copy เก็บทันที

### Revoke API Key

```http
DELETE /api/auth/me/api-key
```

CLI:

```powershell
bun run src/index.ts api-key revoke
```

### API Key Security

```http
PUT /api/auth/me/api-key/security
```

Body:

```json
{
  "allowedIps": ["203.0.113.10"]
}
```

CLI:

```powershell
bun run src/index.ts api-key security --json-file .\api-key-security.json
```

### API Logs

```http
GET /api/auth/me/api-logs?page=1&limit=20
```

CLI:

```powershell
bun run src/index.ts api-key logs --page 1 --limit 20
bun run src/index.ts api-logs --page 1 --limit 20
```

## Webhook API

### List Webhooks

```http
GET /api/auth/me/webhooks
```

CLI:

```powershell
bun run src/index.ts webhook list
```

### Create Webhook

```http
POST /api/auth/me/webhooks
```

Body:

```json
{
  "url": "https://example.com/webhook",
  "events": ["vps.created"]
}
```

CLI:

```powershell
bun run src/index.ts webhook create --url https://example.com/webhook --events vps.created
```

### Delete Webhook

```http
DELETE /api/auth/me/webhooks/{id}
```

CLI:

```powershell
bun run src/index.ts webhook delete <webhook_id>
```

## Interactive Mode

เปิดเมนู interactive:

```powershell
bun run src/index.ts
```

Flow ที่รองรับ:

- VPS: list, create, manage, plans
- Hosting: plans, deploy, manage
- Billing: transactions, export, topup status
- Tickets: list, create, reply, close
- Webhooks
- API key/security
- Raw request

## Recommended Automation Patterns

### Wait for queued VPS action

```powershell
bun run src/index.ts vps reboot <vps_id> --wait --interval 5 --timeout 900
```

### Select only currently orderable plans

```powershell
bun run src/index.ts vps plans --template-id <template_id> --available-only
```

### Script-friendly JSON

```powershell
bun run src/index.ts vps list --compact
```

### Use raw for new endpoints

```powershell
bun run src/index.ts raw GET /api/auth/vps/<vps_id>/activity
```

## Notes for VPS Availability

- `isOrderable=false` means UI/CLI should disable selecting that plan.
- `availabilityStatus=no_capacity` means currently no VPS host can fit that spec.
- `availabilityStatus=out_of_stock` means host capacity may be enough but IP pool is full.
- `nextIpAvailableAt` is an estimate, not a hard SLA.
- IP is not considered available until expired VPS records are archived and removed from active `xen_vps`.
- The archival flow converts expired VMs to template before releasing the IP back to the pool.

## Security Notes

- Treat API keys like passwords.
- Store keys in environment variables or the CLI config file only.
- Do not commit API keys into git.
- Prefer IP allowlist for automation servers.
- Rotate keys when a machine, CI runner, or staff account is retired.
- Full key values are shown only at creation time.

## CLI Development

Install/run with Bun:

```powershell
bun run src/index.ts --help
bun run check
bun run build
```

Project scripts:

```json
{
  "dev": "bun run src/index.ts",
  "check": "bun build src/index.ts --outdir .check",
  "build": "bun build src/index.ts --compile --outfile dist/drite"
}
```

## Quick Command Reference

```powershell
bun run src/index.ts me
bun run src/index.ts doctor
bun run src/index.ts vps plans --template-id <template_id>
bun run src/index.ts vps create --json-file .\create-vps.json --wait
bun run src/index.ts vps watch <vps_id>
bun run src/index.ts hosting deploy --json-file .\deploy-hosting.json
bun run src/index.ts billing transactions --page 1 --limit 20
bun run src/index.ts ticket create --json-file .\ticket.json
bun run src/index.ts webhook list
bun run src/index.ts raw GET /api/auth/me
```
