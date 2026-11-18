# ประกาศอัปเดต Dritestudio CLI และ Go SDK

Dritestudio CLI และ Go SDK เพิ่มคำสั่งสำหรับลูกค้าและ reseller ให้ตรงกับ API ปัจจุบัน

- **VPS** ส่ง SSH public keys ตอนสร้างหรือ reinstall ได้บน image ที่รองรับ รวมถึงเปลี่ยน keys ขอหรือคืน IPv6 และย้อน VPS กลับไปยัง snapshot
- **Reseller** ใช้ `drite reseller vps plans`, `drite reseller vps list` และคำสั่งกลุ่ม `reseller vps` เพื่อดู offer สั่งซื้อ และขอราคา Custom VPS คำสั่ง reseller เดิมยังใช้ได้
- **Hosting** ใช้ `drite hosting plans --account` เพื่อดูราคาตามบัญชี รวมส่วนลด reseller ตามสิทธิ์ และดูสถานะอัปเกรด การใช้งาน ฐานข้อมูล อีเมล และเว็บไซต์จาก CLI
- **Ticket และบัญชี** ปิด Ticket ของตัวเอง ขอ URL อัปโหลดไฟล์ ดูและทำเครื่องหมายประกาศว่าอ่านแล้ว รวมถึงดูรายการและทดสอบ webhook ได้จาก CLI

หากสั่ง VPS ผ่าน Reseller API ให้ตั้ง SSH keys หลัง provisioning ด้วย `drite vps ssh-keys` บน VPS ของบัญชีและ image ที่รองรับ หลังเปลี่ยน SSH keys หรือ IPv6 ให้ตรวจ `rebootRequired` ในผลลัพธ์

เมื่อเผยแพร่รุ่นนี้แล้ว ให้อัปเดต CLI ด้วยคำสั่งต่อไปนี้

```sh
go install github.com/DriteStudio/drite-cli/cmd/drite@latest
```

ดูรายชื่อคำสั่งและตัวอย่างเพิ่มเติมใน [คู่มือ CLI และ Go SDK](../Docs.md) และ [คู่มือ HTTP API](API.md)
