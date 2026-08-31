# FAQ

**"Apakah tool ini bisa memperbaiki konflik secara otomatis?"**
Tidak. `scan --fix` cuma mencetak saran manual dan (setelah konfirmasi `y`
eksplisit) menulis saran itu ke `suggested-fixes.md` sendiri - tidak
pernah mengedit file plugin/skill orang lain. Menerapkan saran itu tetap
langkah manual yang Anda lakukan sendiri.

**"Apakah data saya dikirim ke server mana pun?"**
Secara default: tidak. Satu pengecualian eksplisit: `scan --vuln-check`
mengirim nama+versi dependency (BUKAN isi file, BUKAN kode sumber) ke
osv.dev untuk dicocokkan dengan database kerentanan publik - dan itu
cuma jalan kalau Anda pakai flag itu SENDIRI, dengan prompt konfirmasi
`[y/N]` yang selalu muncul dulu, setiap kali, tidak ada cara
mematikan prompt-nya. Tanpa flag itu, semua scan lokal murni. Lihat
[docs/SECURITY_MODEL.md](SECURITY_MODEL.md).

**"Kenapa tidak dukung Cursor/Codex di v0.1?"**
Fokus satu platform dulu supaya kualitas deteksi tinggi, ekspansi
direncanakan di roadmap v0.3 (lihat [docs/ROADMAP.md](ROADMAP.md)).

**"Apakah estimasi token akurat 100%?"**
Tidak, ini aproksimasi rasio karakter (~4 karakter/token). Untuk angka
presisi, fitur `--exact` (roadmap) akan memanggil API resmi dengan izin
eksplisit.

**"Apakah tool ini scan isi memori/percakapan saya?"**
Tidak pernah. Hanya metadata file (ukuran, waktu, permission) - lihat
`internal/memoryaudit`'s tests untuk bukti langsung (fixture yang sengaja
menaruh "data rahasia" dan memverifikasi itu tidak pernah muncul di output).

**"Kenapa Windows Defender/SmartScreen memperingatkan binary hasil release?"**
Ini pola false-positive yang umum untuk tool baru, kecil, dan belum
code-signed - bukan indikasi tool ini benar-benar berbahaya. Alasannya:
SmartScreen menilai reputasi berdasarkan riwayat unduhan/sertifikat, dan
binary baru otomatis belum punya riwayat itu. Cara paling aman menghindari
peringatan ini sama sekali: install lewat `go install
github.com/damta8827773/agent-stack-audit/cmd/agent-stack-audit@latest`
(kompilasi dari source di mesin sendiri, bukan unduh binary jadi) - ini
memang jalur instalasi utama yang direkomendasikan README. Code-signing
binary release adalah opsi masa depan (butuh sertifikat berbayar), dicatat
di [docs/ROADMAP.md](ROADMAP.md), belum ada di v0.1.

**"Apakah `--vuln-check` sama dengan scan malware/antivirus?"**
Tidak. Ini murni cocokkan versi package (dari `go.mod`/`package.json`/
`requirements.txt`) terhadap database advisory publik OSV.dev/GitHub
Advisory - sama persis teknik yang dipakai `npm audit`/`pip-audit`/
`govulncheck`. Tidak ada disassembly, tidak ada eksekusi kode, tidak ada
deteksi zero-day (kerentanan yang belum dipublikasikan tidak mungkin
terdeteksi lewat pencocokan versi, secara definisi). Lihat
[docs/LIMITATIONS.md](LIMITATIONS.md) untuk batasan lengkapnya.
