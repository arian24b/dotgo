# راهنمای استفاده از dotgo

[English](USAGE.md) · [فارسی](USAGE.fa.md)

## ۱. مفاهیم
- **مخزن** — یک مخزن *bare* گیت (پیش‌فرض `~/.dotfiles`) که work tree آن پوشهٔ home توست. فایل‌های untracked پنهان‌اند، پس فقط چیزی که `add` کنی مدیریت می‌شود.
- **فایل مشترک** — یک فایل tracked معمولی مثل `.gitconfig` که روی همهٔ سیستم‌ها یکسان است.
- **Variant** — فایل هم‌پوشه با پسوند `##`. dotgo برای هر سیستم یکی را انتخاب می‌کند:
  | نام | چه وقتی استفاده می‌شود |
  |---|---|
  | `.zshrc##host.<name>` | نام این سیستم `<name>` باشد (اولویت با این است) |
  | `.zshrc##os.Darwin` / `.zshrc##os.Linux` | سیستم‌عامل بخواند |
  | `.zshrc` | فایل پایه؛ جایگزین و مقصد اعمال |

  کلید `A` نسخهٔ مناسب را روی فایل پایه کپی می‌کند (یا با `--link` / `link_variants` symlink می‌سازد). فایل قبلی با نام `.zshrc.dotgo-bak` می‌ماند.
- **نام سیستم** — فیلد `host` در config، وگرنه hostname سیستم‌عامل.

## ۲. راه‌اندازی
**ماشین اول**
```sh
dotgo init --first --remote git@github.com:YOU/dotfiles.git   # remote اختیاری است
dotgo                       # a ← مسیر فایل (مثلاً .zshrc) برای شروع tracking؛ c ← commit؛ p ← push
```
**ماشین‌های بعدی**
```sh
dotgo init --remote git@github.com:YOU/dotfiles.git --host ubuntu
```
این دستور مخزن را clone می‌کند، فایل‌هایی را که بازنویسی می‌شدند به `~/.dotgo-backup/<زمان>/` می‌برد، checkout می‌کند، upstream را تنظیم می‌کند و پیشنهاد می‌دهد variantهای همین سیستم را اعمال کند.
بدون فلگ، `dotgo init` سؤال می‌پرسد؛ `--yes` مقادیر پیش‌فرض را می‌پذیرد.

**از قبل `~/.dotfiles` داری؟** فقط `dotgo` را اجرا کن — init لازم نیست.

## ۳. کار روزمره
1. `dotgo` ← تب **Changes** فایل‌های تغییرکرده را گروه‌بندی‌شده نشان می‌دهد (نقطهٔ سبز = stage شده، زرد = نشده).
2. با `j`/`k` انتخاب کن، `e` برای ویرایش، `s` برای حالت دو ستونی.
3. `space` فایل را stage می‌کند؛ `]`/`[` یک hunk را انتخاب و `S` فقط همان را stage می‌کند.
4. `c` برای commit و `p` برای push. روی ماشین دیگر: `P` برای pull و بعد `A` برای اعمال دوبارهٔ variantها.

### تب‌ها
| تب | چه نشان می‌دهد |
|---|---|
| 1 Changes | تغییرات working tree، hunkها و staging |
| 2 Upstream | فایل‌هایی که با شاخهٔ remote فرق دارند |
| 3 Variants | فایل‌های دارای `##`؛ diff بین جفت‌ها (`n` = جفت بعدی)؛ نسخهٔ فعال نمایش داده می‌شود |
| 4 All | همهٔ فایل‌های tracked با پیش‌نمایش |
| 5 Log | تاریخچه (`l` = همین فایل، `L` = همه)؛ با `m` یک commit را علامت بزن و commit دیگری را انتخاب کن تا diff دو تا را ببینی |
| 6 Remote | `o` فایل را از `ssh_hosts` می‌گیرد و با نسخهٔ محلی diff می‌کند |

### کلیدها
داخل برنامه `?` را بزن تا فهرست کامل را ببینی. ماوس: wheel اسکرول می‌کند و کلیک ردیف‌ها و تب‌ها را انتخاب می‌کند.

## ۴. تنظیمات
`<user config dir>/dotgo/config.json` — لینوکس: `~/.config/dotgo/`، مک: `~/Library/Application Support/dotgo/` — یا مسیری که در `$DOTGO_CONFIG` بدهی. مخصوص هر ماشین است و داخل مخزن ذخیره نمی‌شود.
```json
{
  "git_dir": "/home/me/.dotfiles",
  "work_tree": "/home/me",
  "host": "ubuntu",
  "remote": "git@github.com:YOU/dotfiles.git",
  "ssh_hosts": ["me@mac.local"],
  "link_variants": false,
  "editor": "nvim"
}
```
فلگ‌ها روی config اولویت دارند: `--config --git-dir --work-tree --ssh --link`. دستور `dotgo config` مقادیر مؤثر را چاپ می‌کند.

## ۵. ایمنی
- Discard، اعمال variant و «stage کردن فایل شبیه secret» همیشه پیش‌نمایش نشان می‌دهند و منتظر `y` می‌مانند.
- نام‌هایی مثل `id_*`، `.env`، `*.pem`، `*token*` و `.gnupg/*` قبل از stage هشدار می‌دهند. کلیدهای خصوصی را از مخزن دور نگه دار.
- pull ناموفق، rebase را abort می‌کند و خطا را نشان می‌دهد؛ چیزی نیمه‌کاره نمی‌ماند.
- تب Remote فقط ssh با کلید را استفاده می‌کند (بدون پرسیدن پسورد) و فقط فایل می‌خواند.

## ۶. رفع اشکال
| مشکل | راه‌حل |
|---|---|
| `no dotfiles repo at …` | `dotgo init` را اجرا کن یا `git_dir` را در config بررسی کن |
| فایل جدید در Changes دیده نمی‌شود | فایل‌های untracked عمداً پنهان‌اند؛ `a` را بزن و مسیرش را وارد کن |
| تب Upstream: «no upstream» | یک بار push کن (`p`) — اولین push، upstream را تنظیم می‌کند |
| variant اعمال نشد | نام را بررسی کن (`##os.Darwin`، `##host.<name>`) و مطمئن شو commit شده؛ تب Variants نسخهٔ فعال را نشان می‌دهد |
| تب Remote: «unavailable» | `ssh user@host` باید بدون پسورد کار کند و ساختار `$HOME` یکسان باشد |
