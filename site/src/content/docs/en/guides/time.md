---
title: Convert Time And Timezones
description: Convert Unix timestamps and common date strings and view multiple timezones.
---

## Supported Input

`leo time` accepts Unix seconds, Unix milliseconds, and common date-time strings. With no value, it uses the current time.

```bash
leo time
leo time 1783512043
leo time 1783512043000
leo time "(2026-07-08 20:00:43)"
```

A numeric value with at least 13 digits is parsed as milliseconds; shorter values are seconds. Dates support `-` or `/` separators, optional seconds, RFC 3339, and forms with an explicit offset.

## Input And Output Timezones

`--from` controls the input timezone for a date string without an explicit timezone, while `--to` controls the primary output timezone. Both accept a UTC offset or IANA name and default to `+8`:

```bash
leo time "2026-08-06 22:26:09" --from -4
leo time 1783512043 --to +9
leo time "2026-07-08 20:00:43" --to Asia/Tokyo
```

The first example interprets the date in UTC-4 and prints the same instant as `2026-08-07 10:26:09` in UTC+8. Unix timestamps and dates with an explicit timezone are not affected by `--from`. Entries in `time.zones` add common-timezone rows; a zone equal to the primary output is not printed twice.
