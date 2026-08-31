# FAQ

**"Apakah tool ini bisa memperbaiki konflik secara otomatis?"**
Tidak di v0.1. Read-only. `--fix` masuk roadmap v0.2+ dengan konfirmasi
interaktif per perubahan, tidak pernah otomatis/silent.

**"Apakah data saya dikirim ke server mana pun?"**
Tidak. Semua scan lokal, tanpa network call default. Lihat
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
