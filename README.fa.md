# dotgo

[English](README.md) · [فارسی](README.fa.md)

یک رابط ترمینالی برای مدیریت فایل‌های dotfiles بین چند سیستم (macOS، Ubuntu و …) داخل یک مخزن git.
بعضی فایل‌ها بین همهٔ سیستم‌ها مشترک‌اند، بعضی برای هر سیستم‌عامل یا هر دستگاه نسخهٔ جدا دارند، و می‌توانی همه را بدون خروج از ترمینال مقایسه کنی.
ظاهر برنامه از [bast.sh](https://bast.sh) الهام گرفته است.

## امکانات
- **نسخه‌های مخصوص هر OS یا هر دستگاه** — `.zshrc##os.Darwin`، `.zshrc##os.Linux`، `.zshrc##host.fpc`؛ با یک کلید نسخهٔ درست همین سیستم اعمال می‌شود.
- **مقایسه** — diff دو ستونی یا یکپارچه بین نسخه‌ها، نسبت به upstream، یا بین هر دو commit.
- **گیت بدون دردسر** — stage فایل یا فقط یک hunk، commit، push، pull (با `--rebase --autostash`) و discard همراه با پیش‌نمایش.
- **امن به‌صورت پیش‌فرض** — کارهای مخرب اول پیش‌نمایش می‌دهند و `y/n` می‌پرسند؛ فایل‌های شبیه credential قبل از stage تأیید می‌گیرند؛ فایل جایگزین‌شده با نام `.dotgo-bak` نگه داشته می‌شود.
- **ویرایش درجا** — کلید `e` فایل را در `$VISUAL` / `$EDITOR` / `vi` باز می‌کند.
- **راه‌اندازی چندسیستمی** — `dotgo init` روی ماشین اول مخزن را می‌سازد و روی ماشین‌های بعدی به آن می‌پیوندد و فایل‌های متعارض را backup می‌گیرد.
- **SSH اختیاری** — diff یک فایل با نسخهٔ همان فایل روی ماشین دیگر.
- با **bare repo** کار می‌کند (`~/.dotfiles`، روش رایج dotfiles) — نیازی به مهاجرت نیست.

## نصب
فایل مناسب سیستم‌عامل و CPU خودت را از [آخرین release](../../releases/latest) دانلود کن (linux/macOS، amd64/arm64):

```sh
tar xzf dotgo_*_*.tar.gz && sudo mv dotgo_*/dotgo /usr/local/bin/
```

یا از سورس بساز (Go 1.24 به بالا):

```sh
go install github.com/arian24b/dotgo/cmd/dotgo@latest
```

## شروع سریع
```sh
# ماشین اول: ساخت مخزن جدید (origin اختیاری است)
dotgo init --first --remote git@github.com:YOU/dotfiles.git

# ماشین‌های بعدی: clone، backup فایل‌های متعارض، اعمال variantهای همین سیستم
dotgo init --remote git@github.com:YOU/dotfiles.git

dotgo            # باز کردن رابط — با ? همهٔ کلیدها را ببین
```
اگر از قبل bare repo در `~/.dotfiles` داری، فقط `dotgo` را اجرا کن.

توضیح بیشتر در **[راهنمای استفاده](docs/USAGE.fa.md)** ([English](docs/USAGE.md)).

## کلیدها (خلاصه)
| کلید | کار | کلید | کار |
|---|---|---|---|
| `1`–`6` | تب‌ها | `space` | stage / unstage فایل |
| `j` `k` | حرکت | `] [` `S` | انتخاب / stage یک hunk |
| `/` | جستجو | `c` `a` | commit / افزودن فایل جدید |
| `e` | ویرایش با `$EDITOR` | `x` | discard (با تأیید) |
| `s` | دو ستونی | `A` | اعمال variant این سیستم |
| `l` `L` `m` | log / علامت‌گذاری commit | `p` `P` | push / pull |
| `?` | راهنما | `q` | خروج |

## توسعه
```sh
go vet ./... && go test ./...
go run ./cmd/dotgo
```
ساختار پروژه و قواعد مشارکت در [AGENTS.fa.md](AGENTS.fa.md) است. پیام commit و عنوان PR باید از
[Conventional Commits](https://www.conventionalcommits.org/) پیروی کند؛ release و release notes با
[semantic-release](https://github.com/semantic-release/semantic-release) در هر merge به `main` خودکار ساخته می‌شود.

## مجوز
MIT
