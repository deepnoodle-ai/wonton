# Bundled fonts

Both fonts are licensed under the SIL Open Font License 1.1.

| File                      | Font                         | License             |
| ------------------------- | ---------------------------- | ------------------- |
| `Inconsolata-Regular.ttf` | Inconsolata Regular          | `OFL.txt`           |
| `NotoEmoji-Regular.ttf`   | Noto Emoji (monochrome), 400 | `NotoEmoji-OFL.txt` |

`NotoEmoji-Regular.ttf` is a static instance of the variable font in
[google/fonts](https://github.com/google/fonts/tree/main/ofl/notoemoji),
downloaded on 2026-10-28. The static instance is less than half the size of
the variable font. To make it again:

```bash
curl -sSLo NotoEmoji-wght.ttf \
  'https://github.com/google/fonts/raw/main/ofl/notoemoji/NotoEmoji%5Bwght%5D.ttf'
fonttools varLib.instancer NotoEmoji-wght.ttf wght=400 -o NotoEmoji-Regular.ttf
```
