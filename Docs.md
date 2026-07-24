# Drite Studio Customer API: Go SDK and CLI

เอกสารนี้อ้างอิง route ฝั่งผู้ใช้ใน `legacy-backend` โดยตรง และอธิบายชื่อ
function ใน Go SDK ที่ใช้แทนแต่ละ endpoint

สำหรับเอกสาร HTTP API ที่ใช้ได้จากทุกภาษา รวมตัวอย่าง `curl`, request body,
error และข้อจำกัดต่าง ๆ ดูที่ [docs/API.md](./docs/API.md)

## Authentication

API token ออกผ่านหน้าเว็บไซต์และมีรูปแบบ:

```text
dr_live_...
```

Backend รองรับทั้งสอง header:

```http
Authorization: Bearer dr_live_xxx
X-API-Key: dr_live_xxx
```

SDK ใช้ Bearer เป็นค่าเริ่มต้น:

```go
client, err := drite.NewClient(os.Getenv("DRITE_API_KEY"))
```

ถ้าต้องการ `X-API-Key`:

```go
client, err := drite.NewClient(
	token,
	drite.WithAPIKeyHeader(),
)
```

ตั้ง staging หรือ local backend:

```go
client, err := drite.NewClient(
	token,
	drite.WithBaseURL("https://staging.example.com"),
)
```

ข้อจำกัดจาก backend:

- API key ใช้สิทธิ์ของเจ้าของบัญชีและมองเห็นเฉพาะ service ของบัญชีนั้น
- บัญชีที่ยังไม่ยืนยันอีเมลหรือถูกระงับอาจถูกปฏิเสธ
- write request ถูกปิดเมื่อบัญชีอยู่ในสถานะ locked
- IP allowlist รองรับ IP แบบตรงตัวเท่านั้น ไม่รองรับ CIDR
- Reseller API ต้องใช้ key ของบัญชี role `reseller`

## Responses and errors

ทุก function คืน:

```go
response, err := client.VPS.Get(ctx, vpsID)
```

อ่าน JSON:

```go
var result map[string]any
err = response.Decode(&result)
```

Non-2xx response คืน `*drite.APIError` และยังคืน `response` เพื่อให้ระบบลูกค้า
เก็บ raw payload ได้:

```go
var apiErr *drite.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, apiErr.Code, apiErr.Message)
}
```

ตัวอย่าง error:

```json
{
  "message": "Customer already has an active ticket",
  "code": "TICKET_ACTIVE_LIMIT"
}
```

## Account API

| HTTP endpoint | Go function | CLI |
| --- | --- | --- |
| `GET /api/auth/me` | `Account.Profile` | `drite me` |
| `PUT /api/auth/me` | `Account.UpdateProfile` | `account update` |
| `POST /api/auth/resend` | `Account.ResendVerification` | `account resend-verification` |
| `GET /api/auth/totp-secret` | `Account.TOTPSecret` | `account totp-secret` |
| `GET /api/auth/me/recovery-codes` | `Account.RecoveryCodeSummary` | `account recovery-codes` |
| `POST /api/auth/me/recovery-codes` | `Account.GenerateRecoveryCodes` | `account generate-recovery-codes` |
| `GET /api/auth/me/sessions` | `Account.Sessions` | `account sessions` |
| `DELETE /api/auth/me/sessions/{id}` | `Account.RevokeSession` | `account revoke-session` |
| `GET /api/auth/me/passkeys` | `Account.Passkeys` | `account passkeys` |
| `POST /api/auth/me/passkeys/register-options` | `Account.BeginPasskeyRegistration` | `account passkey-options` |
| `POST /api/auth/me/passkeys/register-verify` | `Account.FinishPasskeyRegistration` | `account passkey-register` |
| `DELETE /api/auth/me/passkeys/{id}` | `Account.DeletePasskey` | `account delete-passkey` |
| `PUT /api/auth/me/api-key/security` | `Account.SetAPIKeySecurity` | `account api-key-security` |
| `GET /api/auth/me/api-logs` | `Account.APILogs` | `account api-logs` |
| `GET /api/auth/me/webhooks` | `Account.Webhooks` | `account webhooks` |
| `POST /api/auth/me/webhooks` | `Account.CreateWebhook` | `account create-webhook` |
| `DELETE /api/auth/me/webhooks/{id}` | `Account.DeleteWebhook` | `account delete-webhook` |
| `POST /api/auth/me/api-key` | `Account.CreateAPIKey` | `account create-api-key` |
| `DELETE /api/auth/me/api-key` | `Account.RevokeAPIKey` | `account revoke-api-key` |

การเปลี่ยน API key, webhook, session และ security policy อาจต้องส่ง
`currentPassword` หรือ `totpCode` ใน request body ตาม step-up authentication
ของบัญชี

## VPS API

| HTTP endpoint | Go function | CLI action |
| --- | --- | --- |
| `GET /api/auth/vps` | `VPS.List` | `vps list` |
| `GET /api/auth/vps/plans` | `VPS.Plans` | `vps plans` |
| `GET /api/auth/vps/templates` | `VPS.Templates` | `vps templates` |
| `GET /api/auth/vps/available-ips/{hostId}` | `VPS.AvailableIPs` | `vps available-ips` |
| `GET /api/auth/vps/job/{jobId}` | `VPS.Job` | `vps job` |
| `GET /api/auth/vps/failed` | `VPS.Failed` | `vps failed` |
| `DELETE /api/auth/vps/failed/{id}` | `VPS.AcknowledgeFailed` | `vps ack-failed` |
| `POST /api/auth/vps` | `VPS.Create` | `vps create` |
| `POST /api/auth/vps/custom/quote` | `VPS.QuoteCustom` | `vps custom-quote` |
| `POST /api/auth/vps/custom` | `VPS.CreateCustom` | `vps custom-create` |
| `GET /api/auth/vps/{id}` | `VPS.Get` | `vps get` |
| `GET /api/auth/vps/{id}/status` | `VPS.Status` | `vps status` |
| `GET /api/auth/vps/{id}/stats` | `VPS.Stats` | `vps stats` |
| `GET /api/auth/vps/{id}/activity` | `VPS.Activity` | `vps activity` |
| `GET /api/auth/vps/{id}/upgrade-options` | `VPS.UpgradeOptions` | `vps upgrade-options` |
| `POST /api/auth/vps/{id}/upgrade` | `VPS.Upgrade` | `vps upgrade` |
| `GET /api/auth/vps/{id}/snapshots` | `VPS.Snapshots` | `vps snapshots` |
| `POST /api/auth/vps/{id}/snapshots` | `VPS.CreateSnapshot` | `vps snapshot-create` |
| `DELETE /api/auth/vps/{id}/snapshots/{snapshotId}` | `VPS.DeleteSnapshot` | `vps snapshot-delete` |
| `POST /api/auth/vps/{id}/renew` | `VPS.Renew` | `vps renew` |
| `POST /api/auth/vps/{id}/auto-renewal` | `VPS.SetAutoRenewal` | `vps auto-renewal` |
| `POST /api/auth/vps/{id}/rename` | `VPS.Rename` | `vps rename` |
| `POST /api/auth/vps/{id}/reinstall` | `VPS.Reinstall` | `vps reinstall` |
| `POST /api/auth/vps/{id}/control` | `VPS.Control` | ใช้ function โดยตรง |
| `POST /api/auth/vps/{id}/start` | `VPS.Start` | `vps start` |
| `POST /api/auth/vps/{id}/stop` | `VPS.Stop` | `vps stop` |
| `POST /api/auth/vps/{id}/reboot` | `VPS.Reboot` | `vps reboot` |
| `POST /api/auth/vps/{id}/force-stop` | `VPS.ForceStop` | `vps force-stop` |
| `POST /api/auth/vps/{id}/network-reset` | `VPS.NetworkReset` | `vps network-reset` |
| `POST /api/auth/vps/{id}/reset-password` | `VPS.ResetPassword` | `vps reset-password` |
| `DELETE /api/auth/vps/{id}` | `VPS.Delete` | `vps delete` |

สร้าง VPS:

```json
{
  "name": "production",
  "templateId": "template_id",
  "planId": "plan_id",
  "durationType": "monthly",
  "password": "StrongPass1"
}
```

```powershell
drite vps create --data-file .\create-vps.json
```

หลาย operation ตอบ `202` พร้อม `jobId`; ใช้ `VPS.Job` หรือ `drite vps job`
ตรวจจน status เป็น terminal state

## Hosting API

| HTTP endpoint | Go function | CLI action |
| --- | --- | --- |
| `GET /api/un_auth/hosting/plans` | `Hosting.Plans` | `hosting plans` |
| `GET /api/auth/hosting/check-domain` | `Hosting.CheckDomain` | `hosting check-domain` |
| `POST /api/auth/hosting/verify-domain` | `Hosting.VerifyDomain` | `hosting verify-domain` |
| `GET /api/auth/hosting/list` | `Hosting.List` | `hosting list` |
| `GET /api/auth/hosting/{id}` | `Hosting.Get` | `hosting get` |
| `POST /api/auth/hosting/deploy` | `Hosting.Deploy` | `hosting deploy` |
| `POST /api/auth/hosting/{id}/access` | `Hosting.Access` | `hosting access` |
| `POST /api/auth/hosting/{id}/renew` | `Hosting.Renew` | `hosting renew` |
| `GET /api/auth/hosting/{id}/activation-status` | `Hosting.ActivationStatus` | `hosting activation-status` |
| `GET /api/auth/hosting/{id}/activity` | `Hosting.Activity` | `hosting activity` |
| `POST /api/auth/hosting/{id}/autorenew` | `Hosting.ToggleAutoRenewal` | `hosting toggle-auto-renewal` |
| `GET /api/auth/hosting/{id}/stats` | `Hosting.Stats` | `hosting stats` |
| `GET /api/auth/hosting/{id}/disk` | `Hosting.Disk` | `hosting disk` |
| `GET /api/auth/hosting/{id}/traffic` | `Hosting.Traffic` | `hosting traffic` |
| `POST /api/auth/hosting/{id}/reset-password` | `Hosting.ResetPassword` | `hosting reset-password` |
| `DELETE /api/auth/hosting/{id}` | `Hosting.Delete` | `hosting delete` |

`Hosting.Deploy` ใช้ `duration` เป็นจำนวนวัน: 1, 7, 30 หรือ 365

```json
{
  "planId": "plan_id",
  "duration": 30,
  "domain": "example.com",
  "password": "StrongPassw0rd!"
}
```

## Billing API

| HTTP endpoint | Go function | CLI action |
| --- | --- | --- |
| `GET /api/auth/transactions` | `Billing.Transactions` | `billing transactions` |
| `GET /api/auth/transactions/export` | `Billing.ExportTransactions` | `billing export` |
| `GET /api/auth/billing/due-items` | `Billing.DueItems` | `billing due` |
| `POST /api/auth/billing/due-items/pay` | `Billing.PayDueItem` | `billing pay-due` |
| `GET /api/auth/topup/history` | `Billing.TopupHistory` | `billing topup-history` |
| `GET /api/auth/topup/status/{referenceNo}` | `Billing.TopupStatus` | `billing topup-status` |
| `GET /api/auth/biller/signed-url` | `Billing.SignedDocumentURL` | `billing document-url` |

## Ticket API

| HTTP endpoint | Go function | CLI action |
| --- | --- | --- |
| `GET /api/auth/ticket/list` | `Tickets.List` | `ticket list` |
| `GET /api/auth/ticket/{id}` | `Tickets.Get` | `ticket get` |
| `GET /api/auth/ticket/{id}/updates` | `Tickets.Updates` | `ticket updates` |
| `POST /api/auth/ticket` | `Tickets.Create` | `ticket create` |
| `POST /api/auth/ticket/{id}/reply` | `Tickets.Reply` | `ticket reply` |
| `POST /api/auth/ticket/upload-url` | `Tickets.PresignAttachment` | ใช้ function โดยตรง |
| `POST /api/auth/ticket/upload` | `Tickets.UploadAttachment` | `ticket upload` |

ลูกค้าไม่สามารถปิด Ticket เองได้ เจ้าหน้าที่ต้องปิดผ่าน backoffice เพื่อปลด
one-active-ticket gate ดังนั้น SDK ใหม่ตั้งใจไม่เปิด `Close` function

หนึ่งบัญชีมี Ticket ที่ยังไม่ `closed` ได้หนึ่งใบ หาก API ตอบ
`TICKET_ACTIVE_LIMIT` ให้ใช้ `activeTicketId` เปิด Ticket เดิม

Upload flow:

```powershell
drite ticket upload .\debug.log
```

นำ `key` ที่ได้ไปใส่ใน `attachments`:

```json
{
  "subject": "VPS offline",
  "category": "technical",
  "priority": "urgent",
  "message": "see attached log",
  "attachments": ["tickets/user_id/object.log"]
}
```

ข้อจำกัด: TXT, LOG, `image/*`, สูงสุด 5 ไฟล์ต่อข้อความและ 10 MiB ต่อไฟล์

## Reseller API

ใช้ได้เฉพาะ API key ของบัญชี reseller:

| HTTP endpoint | Go function | CLI |
| --- | --- | --- |
| `GET /api/reseller/vps/plans` | `Reseller.VPSPlans` | `reseller vps-plans` |
| `GET /api/reseller/vps/templates` | `Reseller.VPSTemplates` | `reseller vps-templates` |
| `GET /api/reseller/vps` | `Reseller.VPSList` | `reseller vps-list` |
| `GET /api/reseller/vps/{id}` | `Reseller.VPS` | `reseller vps-get` |
| `POST /api/reseller/vps` | `Reseller.CreateVPS` | `reseller vps-create` |
| `POST /api/reseller/vps/custom/quote` | `Reseller.QuoteCustomVPS` | `reseller custom-quote` |
| `POST /api/reseller/vps/custom` | `Reseller.CreateCustomVPS` | `reseller custom-create` |

## Public API

Public functions never attach the customer token:

| HTTP endpoint | Go function | CLI |
| --- | --- | --- |
| `GET /api/un_auth/plans/all` | `Public.Plans` | `public plans` |
| `GET /api/un_auth/hosting/plans` | `Public.HostingPlans` | `public hosting-plans` |
| `GET /api/un_auth/articles/categories` | `Public.ArticleCategories` | `public article-categories` |
| `GET /api/un_auth/articles` | `Public.Articles` | `public articles` |
| `GET /api/un_auth/articles/{slug}` | `Public.Article` | `public article` |

## Raw requests

เมื่อ backend เพิ่ม endpoint ใหม่ก่อน SDK ออกรุ่นใหม่:

```powershell
drite request GET /api/auth/me
drite request GET /api/auth/vps --query take=20 --query skip=0
drite request POST /api/auth/vps/<id>/rename --data '{"name":"new-name"}'
```

Go:

```go
response, err := client.Request(
	ctx,
	http.MethodGet,
	"/api/auth/me",
	nil,
	nil,
)
```

Client ปฏิเสธ absolute endpoint เพื่อไม่ให้ token รั่วไปยัง host อื่น

## CLI configuration

ลำดับ token:

1. global `--token`
2. `DRITE_API_KEY`
3. config จาก `drite auth login`

ลำดับ base URL:

1. global `--base-url`
2. `DRITE_API_URL`
3. config จาก `drite config set-url`
4. `https://dritestudio.co.th`

Global flag ต้องวางก่อน command:

```powershell
drite --compact --timeout 45s vps list
```

## Build and test

```powershell
gofmt -w cmd drite internal
go test ./...
go vet ./...
go build -trimpath -o dist/drite.exe ./cmd/drite
```
