# Source Timezone For `leo time`

## Goal

Allow a timezone-less date string to declare its source timezone:

```bash
leo time "2026-08-06 22:26:09" --from -4
```

With the existing default output timezone of UTC+8, the primary output is:

```text
Time: 2026-08-07 10:26:09 UTC+8
Unix seconds: 1786069569
Unix milliseconds: 1786069569000
```

The real command keeps its existing Chinese output labels. The English labels
above describe the expected values without changing the output contract.

## CLI Contract

Add a `--from` string flag to `leo time`.

- Default: `+8`, preserving the current interpretation of timezone-less dates.
- Accepted values: the same fixed UTC offsets and IANA names accepted by
  `--to`, such as `-4`, `+09:30`, and `America/New_York`.
- Scope: `--from` affects only date strings that do not contain a timezone.
- Unix seconds and milliseconds represent absolute instants and ignore
  `--from`.
- RFC3339 and other inputs with an explicit offset keep that offset and ignore
  `--from`.
- Omitting the input value still uses the current time. An explicitly invalid
  `--from` value is rejected even though no source conversion is needed.

The command always validates `--from`. In the rules above, "ignore" means a
valid source timezone does not change the parsed instant; it does not permit an
invalid flag value.

The existing `--to` flag, its `+8` default, configured common-timezone output,
and all output formatting remain unchanged.

## Implementation

Keep the current command structure and reuse the existing timezone parser.

1. Add a `timeFromZone` flag variable next to `timeToZone`.
2. Register `--from` with a default of `+8` in `timeCmd`.
3. Pass both source and target timezone strings into `runTime`.
4. Parse the source timezone with `parseTimeZone` and pass its location to
   `parseTimeValue` instead of the current hard-coded `fixedZone(8)`.
5. Keep the existing parsing order in `parseTimeValue`: numeric timestamps,
   explicit-offset formats, then timezone-less formats using the supplied
   location.

This requires no new package, dependency, configuration field, or options
abstraction.

## Errors

Invalid source timezones use the existing `parseTimeZone` validation and error
messages. Unsupported date strings retain the existing error behavior. The
command writes no partial output when source parsing fails.

## Tests

Extend `cmd/time_test.go` through the existing `runTime` seam:

- The requested UTC-4 input converts to `2026-08-07 10:26:09 UTC+8` and emits
  the corresponding Unix second and millisecond values.
- An IANA source timezone is accepted.
- An explicit offset in the input takes precedence over `--from`.
- An invalid source timezone returns an error without output.
- Existing timestamp, current-time, target-zone, and configured-zone behavior
  remains green after updating the `runTime` call sites.

Follow red-green-refactor: add the requested behavior test and observe its
expected failure before changing production code.

## Documentation

Update both hand-authored time guides to document `--from`, its default, and
the requested example. Regenerate the Chinese and English command-reference
pages with the repository documentation generator; do not edit generated
reference pages by hand.

## Out Of Scope

- Natural-language dates such as `yesterday` or `next Friday`.
- Timezone abbreviations such as `EST` or `CST`.
- Changes to configured common output zones.
- Changes to the existing output language or layout.
