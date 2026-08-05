# Dritestudio Customer HTTP API

เอกสารนี้สำหรับลูกค้าที่ต้องการเรียก Dritestudio API จากระบบของตัวเองด้วย
API token รูปแบบ `dr_live_...`

Base URL:

```text
https://dritestudio.co.th
```

API ในเอกสารนี้เป็นสิทธิ์ระดับลูกค้าหรือ Reseller เท่านั้น ไม่มี endpoint ระดับ
Admin และทุก resource ที่มีเจ้าของจะถูกตรวจ ownership โดย backend

## เริ่มต้นใช้งาน

สร้าง API token จากหน้าเว็บไซต์ แล้วส่ง token ด้วย Bearer header:

```bash
curl "https://dritestudio.co.th/api/auth/me" \
  -H "Authorization: Bearer dr_live_xxx" \
  -H "Accept: application/json"
```

Backend รองรับ `X-API-Key` เช่นกัน:

```bash
curl "https://dritestudio.co.th/api/auth/me" \
  -H "X-API-Key: dr_live_xxx" \
  -H "Accept: application/json"
```

สำหรับ request ที่มี JSON body ให้เพิ่ม:

```http
Content-Type: application/json
```

ห้ามใส่ token ใน URL, query string, source code, log หรือ repository สาธารณะ

## สิทธิ์และข้อกำหนดของบัญชี

- `/api/auth/*` ใช้สิทธิ์ของเจ้าของ token เท่านั้น
- บัญชีต้องยืนยันอีเมลก่อนใช้ VPS, Hosting, Billing และ Ticket
- บัญชีที่ถูกล็อกยังอ่านข้อมูลที่อนุญาตได้ แต่ write request จะถูกปฏิเสธ
- IP allowlist รองรับเฉพาะ IPv4/IPv6 แบบตรงตัว ไม่รองรับ CIDR
- `/api/reseller/*` ต้องใช้ token ของบัญชี role `reseller`
- `/api/un_auth/*` เป็น Public API และไม่ต้องส่ง token
- ไม่มี Customer API สำหรับปิด Ticket เจ้าหน้าที่ต้องปิดผ่าน Backoffice

## รูปแบบ Response และ Error

Response สำเร็จเป็น JSON แต่ชื่อ envelope อาจเป็น `data`, `active`, `items`,
`transactions` หรือ object โดยตรงตาม endpoint จึงควรอ่านเฉพาะ field ที่ระบบต้องใช้
และรองรับ field ใหม่ในอนาคต

Error ทั่วไป:

```json
{
  "message": "VPS not found",
  "code": "OPTIONAL_MACHINE_CODE"
}
```

HTTP status ที่ควรรองรับ:

| Status | ความหมาย |
| --- | --- |
| `200`, `201` | สำเร็จ |
| `202` | รับงานแล้ว ให้ติดตาม job/operation ต่อ |
| `204` | สำเร็จและไม่มี response body |
| `400` | request หรือข้อมูลไม่ถูกต้อง |
| `401` | token ไม่ถูกต้อง หมดอายุ หรือ role ไม่ตรง |
| `402` | ยอดเงินหรือสถานะ Billing ไม่พร้อม |
| `403` | ไม่มีสิทธิ์ อีเมลยังไม่ยืนยัน บัญชีล็อก หรือ IP ไม่อยู่ใน allowlist |
| `404` | ไม่พบ resource หรือ resource ไม่ใช่ของบัญชีนี้ |
| `409` | resource กำลังทำงานหรือสถานะขัดแย้ง |
| `429` | ถูกจำกัดการใช้งาน เช่น มี Ticket ที่ยังไม่ปิด |
| `500`, `503` | ระบบภายในหรือบริการปลายทางไม่พร้อม |

ถ้า response มี `X-Request-ID` ควรเก็บค่านี้ไว้สำหรับแจ้ง Support

## Account

| Method | Path | รายละเอียด |
| --- | --- | --- |
| `GET` | `/api/auth/me` | ข้อมูลบัญชีและสถานะ API key |
| `PUT` | `/api/auth/me` | แก้โปรไฟล์ รหัสผ่าน หรือ 2FA |
| `POST` | `/api/auth/resend` | ส่งอีเมลยืนยันใหม่ |
| `GET` | `/api/auth/totp-secret` | ขอ secret สำหรับตั้งค่า TOTP |
| `GET` | `/api/auth/me/recovery-codes` | ดูจำนวน Recovery code ที่เหลือ |
| `POST` | `/api/auth/me/recovery-codes` | สร้าง Recovery code ชุดใหม่ |
| `GET` | `/api/auth/me/sessions` | รายการ session |
| `DELETE` | `/api/auth/me/sessions/{sessionId}` | ยกเลิก session |
| `GET` | `/api/auth/me/passkeys` | รายการ Passkey |
| `POST` | `/api/auth/me/passkeys/register-options` | เริ่มลงทะเบียน Passkey |
| `POST` | `/api/auth/me/passkeys/register-verify` | ยืนยันการลงทะเบียน Passkey |
| `DELETE` | `/api/auth/me/passkeys/{passkeyId}` | ลบ Passkey |
| `PUT` | `/api/auth/me/api-key/security` | ตั้ง IP allowlist |
| `GET` | `/api/auth/me/api-logs` | API log ล่าสุดและสถิติ 24 ชั่วโมง |
| `GET` | `/api/auth/me/webhooks` | รายการ Webhook |
| `POST` | `/api/auth/me/webhooks` | สร้าง Webhook |
| `DELETE` | `/api/auth/me/webhooks/{webhookId}` | ลบ Webhook |
| `POST` | `/api/auth/me/api-key` | ออก API key ใหม่ |
| `DELETE` | `/api/auth/me/api-key` | ยกเลิก API key |

การแก้ข้อมูลสำคัญใช้ Step-up authentication โดยส่งอย่างใดอย่างหนึ่งตามการตั้งค่าบัญชี:

```json
{
  "currentPassword": "account-password",
  "totpCode": "123456"
}
```

ตั้ง IP allowlist:

```bash
curl -X PUT "https://dritestudio.co.th/api/auth/me/api-key/security" \
  -H "Authorization: Bearer dr_live_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "allowedIps": ["203.0.113.10", "2001:db8::10"],
    "currentPassword": "account-password"
  }'
```

ส่ง `allowedIps: []` เพื่อปิด allowlist การใส่ CIDR เช่น `203.0.113.0/24`
จะถูกปฏิเสธ

สร้าง Webhook:

```json
{
  "url": "https://customer.example/webhooks/drite",
  "events": ["vps.created", "vps.started", "vps.stopped"],
  "secret": "webhook-signing-secret",
  "totpCode": "123456"
}
```

Event ที่รองรับ:

- `vps.created`
- `vps.deleted`
- `vps.started`
- `vps.stopped`
- `vps.renewed`

## VPS

### รายการและข้อมูล

| Method | Path | Query |
| --- | --- | --- |
| `GET` | `/api/auth/vps` | `take`, `skip` |
| `GET` | `/api/auth/vps/plans` | `templateId` ไม่บังคับ |
| `GET` | `/api/auth/vps/templates` | - |
| `GET` | `/api/auth/vps/available-ips/{hostId}` | - |
| `GET` | `/api/auth/vps/job/{jobId}` | - |
| `GET` | `/api/auth/vps/failed` | `take`, `skip` |
| `DELETE` | `/api/auth/vps/failed/{failureId}` | - |
| `GET` | `/api/auth/vps/{vpsId}` | - |
| `GET` | `/api/auth/vps/{vpsId}/status` | - |
| `GET` | `/api/auth/vps/{vpsId}/stats` | - |
| `GET` | `/api/auth/vps/{vpsId}/activity` | - |
| `GET` | `/api/auth/vps/{vpsId}/upgrade-options` | - |
| `GET` | `/api/auth/vps/{vpsId}/snapshots` | - |

สร้าง VPS:

```bash
curl -X POST "https://dritestudio.co.th/api/auth/vps" \
  -H "Authorization: Bearer dr_live_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production",
    "templateId": "template_id",
    "planId": "plan_id",
    "durationType": "monthly",
    "password": "StrongPass1"
  }'
```

`durationType` รองรับ `daily`, `weekly`, `monthly`, `yearly`

รหัสผ่าน VPS ต้องยาว 8-64 ตัว, ห้ามขึ้นต้นด้วยตัวเลข และต้องมีตัวพิมพ์ใหญ่
ตัวพิมพ์เล็ก และตัวเลขอย่างน้อยอย่างละหนึ่งตัว

สร้าง Custom VPS:

```http
POST /api/auth/vps/custom/quote
POST /api/auth/vps/custom
```

Quote body:

```json
{
  "templateId": "template_id",
  "durationType": "monthly",
  "cpu": 4,
  "ram": 8,
  "disk": 100
}
```

Create body ใช้ field เดียวกับ Quote และเพิ่ม `name`, `password`; `templateId`
เป็น field บังคับตอนสร้าง

### VPS actions

| Method | Path | JSON body |
| --- | --- | --- |
| `POST` | `/api/auth/vps/{vpsId}/renew` | `{"durationType":"monthly"}` |
| `POST` | `/api/auth/vps/{vpsId}/auto-renewal` | `{"enabled":true}` |
| `POST` | `/api/auth/vps/{vpsId}/upgrade` | `{"planId":"plan_id"}` |
| `POST` | `/api/auth/vps/{vpsId}/rename` | `{"name":"new-name"}` |
| `POST` | `/api/auth/vps/{vpsId}/reinstall` | `{"templateId":"template_id","password":"StrongPass1"}` |
| `POST` | `/api/auth/vps/{vpsId}/control` | `{"action":"start"}` |
| `POST` | `/api/auth/vps/{vpsId}/start` | ไม่ต้องมี body |
| `POST` | `/api/auth/vps/{vpsId}/stop` | ไม่ต้องมี body |
| `POST` | `/api/auth/vps/{vpsId}/reboot` | ไม่ต้องมี body |
| `POST` | `/api/auth/vps/{vpsId}/force-stop` | ไม่ต้องมี body |
| `POST` | `/api/auth/vps/{vpsId}/network-reset` | ไม่ต้องมี body |
| `POST` | `/api/auth/vps/{vpsId}/reset-password` | `{"password":"NewStrongPass1"}` |
| `POST` | `/api/auth/vps/{vpsId}/snapshots` | `{"name":"before-upgrade"}` |
| `DELETE` | `/api/auth/vps/{vpsId}/snapshots/{snapshotId}` | - |
| `DELETE` | `/api/auth/vps/{vpsId}` | - |

`control.action` รองรับ `start`, `stop`, `reboot`, `force-stop`

Operation ที่ตอบ `202` จะมี `jobId` หรือ operation identifier ให้เรียก
`GET /api/auth/vps/job/{jobId}` จนถึง terminal state ไม่ควรยิง action ซ้ำระหว่าง
VPS อยู่ในสถานะทำงาน

## Hosting

| Method | Path | รายละเอียด |
| --- | --- | --- |
| `GET` | `/api/un_auth/hosting/plans` | แพ็กเกจ Public |
| `GET` | `/api/auth/hosting/check-domain?domain=example.com` | ตรวจโดเมน |
| `POST` | `/api/auth/hosting/verify-domain` | ตรวจ TXT ownership |
| `GET` | `/api/auth/hosting/list` | Hosting ของบัญชี |
| `GET` | `/api/auth/hosting/{hostingId}` | รายละเอียด |
| `POST` | `/api/auth/hosting/deploy` | สร้าง Hosting |
| `POST` | `/api/auth/hosting/{hostingId}/access` | ขอ Plesk login URL |
| `POST` | `/api/auth/hosting/{hostingId}/renew` | ต่ออายุ |
| `GET` | `/api/auth/hosting/{hostingId}/activation-status` | สถานะเปิดใช้งาน |
| `GET` | `/api/auth/hosting/{hostingId}/activity` | Activity |
| `GET` | `/api/auth/hosting/{hostingId}/upgrade-options` | แผนที่อัปเกรดได้และราคาส่วนต่าง |
| `POST` | `/api/auth/hosting/{hostingId}/upgrade` | ส่งคำขออัปเกรดเข้าคิว |
| `POST` | `/api/auth/hosting/{hostingId}/autorenew` | สลับเปิด/ปิด Auto-renew |
| `GET` | `/api/auth/hosting/{hostingId}/stats` | สถิติรวม |
| `GET` | `/api/auth/hosting/{hostingId}/disk` | Disk usage |
| `GET` | `/api/auth/hosting/{hostingId}/traffic` | Traffic |
| `POST` | `/api/auth/hosting/{hostingId}/reset-password` | เปลี่ยนรหัสผ่าน |
| `DELETE` | `/api/auth/hosting/{hostingId}` | ลบ Hosting |

ตรวจและยืนยันโดเมน:

```json
{
  "domain": "example.com",
  "token": "verification-token-from-check-domain"
}
```

สร้าง Hosting:

```json
{
  "planId": "plan_id",
  "duration": 30,
  "domain": "example.com",
  "password": "StrongPassw0rd!",
  "domainVerificationToken": "token-if-required"
}
```

`duration` เป็นจำนวนวัน: `1`, `7`, `30` หรือ `365`

อัปเกรด Hosting:

```json
{
  "planId": "plan_id"
}
```

เรียก `GET /api/auth/hosting/{hostingId}/upgrade-options` ก่อนเพื่อรับรายการแผนและ
ราคาส่วนต่างที่ server คำนวณตามวันคงเหลือ จากนั้นส่ง body ข้างต้นไปยัง
`POST /api/auth/hosting/{hostingId}/upgrade` เมื่อรับงานสำเร็จ endpoint จะตอบ `202`
พร้อม `jobId` โดยวันต่ออายุและรอบบิลเดิมไม่เปลี่ยน

รหัสผ่าน Hosting ต้องยาว 10-128 ตัว, ห้ามขึ้นต้นด้วยตัวเลข และต้องมีตัวพิมพ์ใหญ่
ตัวพิมพ์เล็ก ตัวเลข และอักขระพิเศษ

Endpoint `/autorenew` เป็นการ toggle ทุกครั้งที่เรียก ควรอ่านสถานะปัจจุบันจาก
`GET /api/auth/hosting/{hostingId}` ก่อน หากต้องการบังคับ final state

## Billing และ Top-up

| Method | Path | Query/Body |
| --- | --- | --- |
| `GET` | `/api/auth/transactions` | `page`, `limit`, `month`, `startDate`, `endDate` |
| `GET` | `/api/auth/transactions/export` | `month`, `startDate`, `endDate` |
| `GET` | `/api/auth/billing/due-items` | - |
| `POST` | `/api/auth/billing/due-items/pay` | `{"type":"vps","serviceId":"..."}` |
| `GET` | `/api/auth/topup/history` | - |
| `GET` | `/api/auth/topup/status/{referenceNo}` | - |
| `GET` | `/api/auth/biller/signed-url` | `id`, `type=transaction|topup` |

ตัวอย่าง:

```bash
curl "https://dritestudio.co.th/api/auth/transactions?page=1&limit=20&month=2026-07" \
  -H "Authorization: Bearer dr_live_xxx"
```

`billing/due-items/pay` รองรับ `type` เป็น `vps` หรือ `hosting`

## Ticket

ลูกค้าหนึ่งบัญชีมี Ticket ที่สถานะยังไม่ `closed` ได้หนึ่งใบเท่านั้น การสร้างซ้ำ
จะตอบ `429`:

```json
{
  "code": "TICKET_ACTIVE_LIMIT",
  "message": "คุณมี Ticket ที่กำลังดำเนินการอยู่ กรุณารอให้เจ้าหน้าที่ปิด Ticket เดิมก่อน",
  "activeTicketId": "ticket_id",
  "cooldownHours": 1,
  "unlockPolicy": "staff_close"
}
```

ข้อจำกัดนี้ไม่ reset ตามเวลา ลูกค้าจะสร้างใบใหม่ได้ทันทีหลังเจ้าหน้าที่ปิดใบเดิม
ผ่าน Backoffice

| Method | Path | รายละเอียด |
| --- | --- | --- |
| `GET` | `/api/auth/ticket/list` | Query: `page`, `limit`, `status`, `category`, `search` |
| `GET` | `/api/auth/ticket/{ticketId}` | รายละเอียดและข้อความ |
| `GET` | `/api/auth/ticket/{ticketId}/updates` | Query: `updatedSince` แบบ RFC3339 |
| `POST` | `/api/auth/ticket` | สร้าง Ticket |
| `POST` | `/api/auth/ticket/{ticketId}/reply` | ตอบ Ticket |
| `POST` | `/api/auth/ticket/upload` | Multipart upload |
| `POST` | `/api/auth/ticket/upload-url` | ขอ Presigned upload URL |

สร้าง Ticket:

```json
{
  "subject": "VPS offline",
  "category": "technical",
  "priority": "urgent",
  "message": "ไม่สามารถเชื่อมต่อ VPS ได้",
  "serviceType": "vps",
  "serviceId": "vps_id",
  "attachments": []
}
```

ค่าที่รองรับ:

- `category`: `technical`, `billing`, `sales`, `security`, `migration`
- `priority`: `low`, `normal`, `urgent`
- `serviceType`: `vps`, `hosting`
- ถ้าส่ง `serviceType` ต้องส่ง `serviceId` คู่กัน และ service ต้องเป็นของบัญชี
- `subject` ยาวไม่เกิน 200 ตัวอักษร
- `message` ยาวไม่เกิน 10,000 ตัวอักษร

ตอบ Ticket:

```json
{
  "message": "แนบ log เพิ่มเติมแล้ว",
  "attachments": ["tickets/user_id/attachment.log"]
}
```

### อัปโหลดไฟล์ Ticket

รองรับเฉพาะ:

- `.txt`
- `.log`
- MIME type `image/*`

จำกัดไฟล์ละ 10 MiB และสูงสุด 5 ไฟล์ต่อการสร้าง Ticket หรือหนึ่ง reply ไม่รองรับ
PDF, ZIP, executable และไฟล์ชนิดอื่น

Multipart:

```bash
curl -X POST "https://dritestudio.co.th/api/auth/ticket/upload" \
  -H "Authorization: Bearer dr_live_xxx" \
  -F "file=@debug.log;type=text/plain"
```

นำ `key` จาก response ไปใส่ใน `attachments` ห้ามใช้ public URL แทน key

Presigned URL:

```bash
curl -X POST "https://dritestudio.co.th/api/auth/ticket/upload-url" \
  -H "Authorization: Bearer dr_live_xxx" \
  -H "Content-Type: application/json" \
  -d '{"filename":"screenshot.png","mimeType":"image/png"}'
```

อัปโหลดไฟล์ไปยัง URL ที่ backend คืนมา แล้วใช้ object `key` ใน Ticket request
โดยต้องส่ง `Content-Type` ให้ตรงกับค่าที่ใช้ขอ URL

ลูกค้าไม่มี endpoint สำหรับปิด Ticket การเรียก
`PUT /api/auth/ticket/{ticketId}/close` จะตอบ `403 STAFF_CLOSE_REQUIRED`

## Reseller VPS API

ต้องใช้ token ของบัญชี role `reseller` เท่านั้น:

| Method | Path |
| --- | --- |
| `GET` | `/api/reseller/vps/plans` |
| `GET` | `/api/reseller/vps/templates` |
| `GET` | `/api/reseller/vps` |
| `GET` | `/api/reseller/vps/{vpsId}` |
| `POST` | `/api/reseller/vps` |
| `POST` | `/api/reseller/vps/custom/quote` |
| `POST` | `/api/reseller/vps/custom` |

Request body ใช้รูปแบบเดียวกับ Customer VPS แต่ระบบคิดราคาและตรวจสิทธิ์ตามบัญชี
Reseller บัญชีลูกค้าปกติจะถูกปฏิเสธและไม่ได้รับสิทธิ์เพิ่ม

## Public API

Public endpoint ไม่ต้องส่ง token:

| Method | Path | Query |
| --- | --- | --- |
| `GET` | `/api/un_auth/plans/all` | - |
| `GET` | `/api/un_auth/hosting/plans` | - |
| `GET` | `/api/un_auth/articles/categories` | - |
| `GET` | `/api/un_auth/articles` | `page`, `limit`, `category`, `search` |
| `GET` | `/api/un_auth/articles/{slug}` | - |

ไม่ควรส่ง Authorization header ไปยัง Public API

## ใช้ผ่าน Go SDK

ติดตั้ง:

```bash
go get github.com/DriteStudio/drite-cli/drite
```

ตัวอย่าง:

```go
client, err := drite.NewClient(os.Getenv("DRITE_API_KEY"))
if err != nil {
    log.Fatal(err)
}

response, err := client.VPS.List(
    context.Background(),
    drite.VPSListOptions{Take: 20},
)
if err != nil {
    var apiErr *drite.APIError
    if errors.As(err, &apiErr) {
        log.Printf("HTTP %d: %s (%s)", apiErr.StatusCode, apiErr.Message, apiErr.Code)
    }
    log.Fatal(err)
}

var payload map[string]any
if err := response.Decode(&payload); err != nil {
    log.Fatal(err)
}
```

Named function ของทุก endpoint ดูได้ที่ [Go SDK and CLI mapping](../Docs.md)

## ใช้ผ่าน CLI

```powershell
drite auth login --token dr_live_xxx
drite me
drite vps list
drite hosting list
drite ticket list --status open
```

ดูคำสั่งและตัวอย่างทั้งหมดได้จาก [README](../README.md) และ
[Go SDK and CLI mapping](../Docs.md)
