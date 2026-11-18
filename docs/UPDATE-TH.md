# ประกาศอัปเดต Dritestudio CLI และ Go SDK

Dritestudio CLI และ Go SDK เพิ่มคำสั่งสำหรับลูกค้าและ reseller ให้ตรงกับ API ปัจจุบัน

- **VPS** เปลี่ยน SSH keys ขอหรือคืน IPv6 และย้อน VPS กลับไปยัง snapshot ได้จาก CLI
- **SSH keys ในบัญชี** บันทึก SSH public keys ในบัญชีด้วย `drite ssh-key add` แล้วใช้ `--ssh-key` เพื่อเลือกตอนสร้างหรือ reinstall บน image ที่รองรับ ใช้กับคำสั่ง `vps` และ `reseller vps` ได้
- **Reseller** ใช้ `drite reseller vps plans`, `drite reseller vps list` และคำสั่งกลุ่ม `reseller vps` เพื่อดู offer สั่งซื้อ และขอราคา Custom VPS คำสั่ง reseller เดิมยังใช้ได้
- **Hosting** ใช้ `drite hosting plans --account` เพื่อดูราคาตามบัญชี รวมส่วนลด reseller ตามสิทธิ์ และดูสถานะอัปเกรด การใช้งาน ฐานข้อมูล อีเมล และเว็บไซต์จาก CLI
- **Ticket และบัญชี** ปิด Ticket ของตัวเอง ขอ URL อัปโหลดไฟล์ ดูและทำเครื่องหมายประกาศว่าอ่านแล้ว รวมถึงดูรายการและทดสอบ webhook ได้จาก CLI

แต่ละ VPS มีสำเนา SSH keys ของตัวเอง หากต้องการถอน key จากเครื่อง ให้แก้ในหน้า Access หรือใช้ `drite vps ssh-keys` แล้วตรวจ `rebootRequired` ในผลลัพธ์

เมื่อเผยแพร่รุ่นนี้แล้ว ให้อัปเดต CLI ด้วยคำสั่งต่อไปนี้

```sh
go install github.com/DriteStudio/drite-cli/cmd/drite@latest
```

ดูรายชื่อคำสั่งและตัวอย่างเพิ่มเติมใน [คู่มือ CLI และ Go SDK](../Docs.md) และ [คู่มือ HTTP API](API.md)
