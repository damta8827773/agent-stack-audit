# CLAUDE.md - Aturan Proyek agent-stack-audit (OTORITATIF)

> File ini dibaca otomatis oleh Claude Code di setiap sesi kerja pada
> proyek ini. Aturan di sini bersifat MENGIKAT dan berada di atas
> instruksi ad-hoc dalam chat, KECUALI pemilik proyek secara eksplisit
> menulis di prompt: "abaikan aturan CLAUDE.md poin X untuk permintaan
> ini" - tanpa kalimat itu, konflik harus DITUNJUKKAN dulu ke pemilik
> proyek, bukan dieksekusi diam-diam maupun ditolak diam-diam.

## 1. ATURAN KERAS STACK TEKNOLOGI (tidak bisa diubah lewat chat biasa)

- Bahasa: **Go 1.22+ SAJA.**
- **DILARANG**: HTML, CSS, JavaScript, TypeScript, Node.js, atau
  dependency apa pun yang membutuhkan runtime tersebut - di bagian
  MANA PUN proyek ini, termasuk tooling build/dokumentasi.
- Alasan: menghilangkan seluruh kelas risiko supply-chain attack dari
  ekosistem npm secara struktural. Keputusan sadar, bukan kelalaian.
- Permintaan "tambah bahasa X" di masa depan → Claude Code WAJIB
  berhenti dan konfirmasi ulang dulu, jangan langsung eksekusi.

## 2. ATURAN KERAS KEJUJURAN (anti AI-slop, anti klaim palsu)

- TIDAK PERNAH membuat gambar/video yang mengklaim menunjukkan tool ini
  berjalan padahal tidak - termasuk video generative AI (Veo dsb.)
  untuk demo produk. Demo HANYA dari rekaman sesi terminal ASLI (pakai
  `vhs` dari Charm, satu ekosistem dengan lipgloss/bubbletea).
- TIDAK PERNAH menulis klaim keamanan absolut ("tidak bisa diretas
  siapa pun", "100% aman"). Klaim harus spesifik & bisa diverifikasi.
- TIDAK PERNAH mengklaim tim multi-disiplin/internasional kalau
  kenyataannya satu developer.
- Semua angka di README/laporan berasal dari eksekusi nyata, bukan
  karangan.
- Kata terlarang tanpa bukti angka: "revolutionary", "game-changing",
  "seamless", "blazing fast", "next-generation", "cutting-edge",
  "state-of-the-art", "powerful" (tanpa kualifikasi).

## 3. ATURAN KEAMANAN OPERASIONAL & MODUL SENSITIF

**Prinsip umum:**
- Read-only secara default, selalu. Tidak ada modifikasi/hapus file
  pengguna tanpa flag eksplisit `--fix` + konfirmasi interaktif.
- Zero network call tanpa flag eksplisit + peringatan tertulis
  sebelum jalan.

**memory-audit - batasan mutlak:**
- TIDAK PERNAH `SELECT *` isi baris SQLite atau membaca isi file
  memori Markdown. Hanya metadata: path, ukuran, waktu modifikasi,
  permission. Pelanggaran ini bug kritis (P0), bukan bug biasa.

**vuln-audit - sumber data & alur:**
- Sumber: OSV.dev (utama, gratis, tanpa API key) + GitHub Advisory
  Database (sekunder).
- Hanya jalan dengan flag eksplisit `--vuln-check`, WAJIB konfirmasi
  [y/N] sebelum mengirim data (nama+versi package saja) ke osv.dev.
- Severity diambil langsung dari data OSV/GHSA, tool ini TIDAK PERNAH
  menilai severity sendiri.
- BUKAN SAST/DAST - murni pencocokan versi ke advisory publik. Zero-day
  yang belum dipublikasikan tidak terdeteksi (tulis di LIMITATIONS.md).

**Audit log & integritas:**
- `~/.agent-stack-audit/audit-log.jsonl` - append-only, tiap baris
  menyertakan `prev_hash` + `entry_hash` (SHA-256) untuk hash-chain.
  `verify-log` melaporkan baris yang rusak kalau chain putus.
- Retention: rotate ke `.jsonl.gz` setelah 90 hari, jangan pernah hapus.

**Rilis & tamper-resistance:**
- Tiap rilis: `checksums.txt` (SHA-256) ditandatangani via cosign.
  SBOM format CycloneDX disertakan tiap rilis.
- Branch protection wajib di `main`: PR wajib, 1 review approve, CI
  lulus. CODEOWNERS wajib untuk perubahan di internal/memoryaudit/ dan
  internal/vulnaudit/ (2 reviewer).

**Realita yang tidak boleh dibantah:**
- Repo publik tidak bisa "dikunci" dari pembacaan siapa pun (manusia
  atau AI). `.github/AI_AGENT_NOTICE.md` bersifat kebijakan tertulis,
  BUKAN pagar teknis - jangan tulis di mana pun bahwa itu "memblokir".

## 4. VISIBILITY REPO

- Repo boleh diset **Private** kapan saja - kontrol sah dan efektif.
- TIDAK ADA cara membuat repo **Public** "kebal dibaca" siapa pun -
  sifat dasar internet, bukan bug. Jangan pernah tulis klaim sebaliknya.
- Permintaan "blokir akses orang lain" di masa depan → klarifikasi dulu
  maksudnya apa, jangan asumsikan yang mustahil.

## 5. PERTUMBUHAN & RILIS

- Tidak ada bot star, akun palsu, atau skema engagement artifisial
  dalam bentuk apa pun, permanen, tanpa pengecualian.
- Klaim performa/kualitas didukung angka nyata dari eksekusi tool.

## 6. RINGKASAN MODUL PROYEK

| Modul | Tugas satu baris |
|---|---|
| discover | Temukan semua skill/plugin/hook yang aktif |
| conflict-check | Deteksi hook dari sumber berbeda yang bentrok event+matcher |
| token-cost | Estimasi overhead token dari skill always-on |
| memory-audit | Laporkan metadata memori lokal - TIDAK PERNAH baca isinya |
| trust-report | Cek permission, LICENSE, panggilan network mencurigakan |
| vuln-audit | Cocokkan versi dependency ke database CVE publik (opt-in) |

Detail lengkap tiap modul: `docs/ARCHITECTURE.md` (kontrak antar modul,
alur data) dan README.md bagian "What it checks" (perilaku tiap modul
dari sudut pandang pengguna). (Catatan 2026-09-07: baris ini sebelumnya
merujuk file `agent-stack-audit-spec.md` yang ternyata tidak pernah ada
di repo - referensi mati, diperbaiki ke dokumen yang benar-benar ada.)

## 7. ASET VISUAL - ANTI AI-SLOP

**Demo/screenshot:**
- HANYA dari rekaman sesi terminal ASLI (`vhs`), TIDAK PERNAH video
  generative AI untuk demo produk.
- 3 screenshot wajib: scan-output-terminal.png, conflict-example.png,
  report-markdown-sample.png - diambil SETELAH tool benar-benar
  berjalan.

**Kriteria logo (bukan AI slop):**
- Hindari: gradient berlebihan, glossy 3D, ikon "otak sirkuit"/robot
  generik, drop-shadow berlebih di ukuran kecil.
- Disarankan: flat, monokrom/dua warna, geometris, line-icon (kaca
  pembesar/perisai/terminal-cursor). Uji keterbacaan di 16×16px.
- SVG + PNG di docs/assets/, cek lisensi tiap aset satu per satu,
  catat di docs/ASSET_LICENSES.md.

**Font:**
- Screenshot terminal: JetBrains Mono / Fira Code / Cascadia Code
  (semua OFL, lisensi jelas).
- Banner butuh font kustom → render statis PNG, prioritas Google Fonts
  di atas comot acak dari situs font pihak ketiga.
- Tidak ada logo/banner final tanpa review manusia dulu.

## 8. STANDAR & KLAIM KEPATUHAN

**Boleh diklaim (bisa dipenuhi sungguhan):**
- "Designed with alignment to ISO/IEC 27001 Annex A principles" - BUKAN
  "certified" (sertifikasi = proses organisasi, bukan repo kecil).
- Pemetaan modul ke NIST CSF 2.0 (Identify/Protect/Detect/Respond/
  Recover) di docs/ARCHITECTURE.md.
- SPDX-License-Identifier di header tiap file .go. SBOM CycloneDX per
  rilis. SemVer 2.0.0, Keep a Changelog 1.1.0, Conventional Commits
  1.0.0. OWASP CLI Security Cheat Sheet.

**Tidak boleh diklaim:**
- Sertifikasi resmi tanpa proses sungguhan.
- "Tim internasional/multi-disiplin" kalau kenyataannya satu developer.

## 9. ATURAN TESTING

- Semua test pakai fixture di testdata/fixtures/ - TIDAK PERNAH scan
  direktori config asli milik developer/pengguna.
- Target coverage v0.1: minimal 70% untuk internal/discover,
  internal/conflict, internal/trustreport.
- Golden file test untuk report.json. Integration test: fixture
  gabungan dengan konflik hook yang disengaja.
- PR yang mengubah internal/memoryaudit/ atau internal/vulnaudit/ WAJIB
  menyertakan test baru.

## 10. PROTOKOL RESOLUSI KONFLIK

Kalau instruksi baru tampak bertentangan dengan bagian 1–9:
1. Sebutkan bagian mana yang bertentangan.
2. Jelaskan alasan teknisnya singkat.
3. Tawarkan alternatif yang memenuhi TUJUAN di balik permintaan.
4. Eksekusi hanya setelah pemilik proyek menjawab.

## 11. RIWAYAT PERUBAHAN

- 2026-08-31: Draf awal (86 baris)
- 2026-08-31: Sempat dipecah ke CLAUDE.md + .claude/rules/ (4 file)
- 2026-08-31: Digabung ulang jadi satu file utuh atas permintaan
  pemilik proyek (lebih mudah dikelola, trade-off: >200 baris)
- 2026-09-07: Dipindah dari luar repo (`new system/claude.md`, tidak
  ter-version-control) ke `agent-stack-audit/CLAUDE.md` (root repo) -
  sebelumnya file ini mengklaim "dibaca otomatis... checked into the
  codebase" padahal secara nyata tidak berada di dalam git repo mana
  pun. Referensi mati ke `agent-stack-audit-spec.md` di bagian 6 juga
  diperbaiki di perubahan yang sama.