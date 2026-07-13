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

A date string without an explicit timezone is interpreted as UTC+8. `--to` controls the primary output timezone and accepts a UTC offset or IANA name:

```bash
leo time 1783512043 --to +9
leo time "2026-07-08 20:00:43" --to Asia/Tokyo
```

The default is `+8`. Entries in `time.zones` add common-timezone rows; a zone equal to the primary output is not printed twice.
