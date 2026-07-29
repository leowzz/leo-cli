# Batch User SQL Generator Design

## Goal

Add `leo batch-users`, a local command that generates MySQL user inserts and
matching Redis access-token mappings for a contiguous three-digit UID suffix
range. The command only writes text to stdout and never connects to MySQL or
Redis.

## Command

```bash
leo batch-users --uid '3181538941{001,010}' [--token-prefix leo]
```

- `--uid` is required and has the exact form `<numeric-prefix>{NNN,NNN}`.
- Both bounds are three decimal digits and form an inclusive range from `000`
  through `999`.
- The start must not exceed the end.
- `--token-prefix` defaults to `leo` and must be non-empty with no whitespace.
- The UID template must be quoted in the shell so brace expansion does not
  alter it before Cobra receives the value.

For the example above, the generated IDs are `3181538941001` through
`3181538941010`.

## MySQL Output

Write one multi-row `INSERT INTO \`mindcraft_tokyo\`.\`users\`` statement. Use
the column list and values from the supplied source user, with these changes
for each generated suffix:

- `id`: numeric prefix plus the zero-padded three-digit suffix.
- `username`: `leot1u` plus the suffix.
- `email`: `leot1u` plus the suffix plus `@hakko.ai`.
- `created_at` and `updated_at`: `NOW()`.

All other values remain fixed:

```text
hashed_password = $2b$12$ccC5ZsvGLPBYE8OcI3D6qeu/nGQuwIvB1YtnHK185XljLwlSPOJ/a
country_code = ''
phone = NULL
is_phone_verified = 0
is_active = 1
is_superuser = 0
invited_by = 1
invitation_limit = 0
is_email_verified = 1
is_institution = 0
register_device_identifier = 299332346848754281711
character_being_used = 22
is_cyber = 0
character_language = en
source = HakkoAI-v0.5.8.1-Install.exe
active_conversation_frequency = middle
account_type = 1
other_platform_uid = 104907950741872951526
ys_map_search_goods = 0
ys_active_dialogue = 2
bubble_switch = 1
avatar = https://download.hakko.ai/feedback/178410028260afb5d8f9ca461bada4e34bb72dcd35_avatar_file
private_mode_switch = NULL
private_mode_password = NULL
game_assistant_switch = 1
register_platform = 1
mobile_market = ''
sex = 2
birthday = 2025.08.25
system_language = zh
character_being_used_mobile = 64
internet_search_switch = 0
screen_game_vision_switch_pc = 2
screen_game_vision_switch_mobile = 1
ys_active_dialogue_mobile = 1
bubble_switch_mobile = 1
register_info = {"lan":"en","area":"jp","flags":["count","11"],"adjust_id":"47af3382e41167b2885b96bda3d64df8","client_ip":"117.134.14.85","app_version":"1.0.0.2","app_platform":"windows","platform_source":"earlypreview"}
settings = {"timezone":"Asia/Shanghai","cv_tip_switch_pc":1,"user_defined_model":"gemini-3.1-flash-lite","cv_active_push_freq":4,"adult_content_switch":1,"cv_active_push_switch":1,"conversation_mode_mobile":2}
bio = sed in esse date
```

Preserve the source column order. Quote SQL string and JSON values with single
quotes, keep numeric values unquoted, and emit `NULL` without quotes.

## Redis Output

After the complete MySQL statement, emit exactly two empty lines, then one
Redis command per generated account:

```text
set access_token:<token-prefix><suffix>:user_id:app_platform <uid>:web
```

Example:

```text
set access_token:leo001:user_id:app_platform 3181538941001:web
set access_token:leo002:user_id:app_platform 3181538941002:web
```

The separator is three newline characters after the MySQL statement: one ends
the SQL line and two form the empty lines before the first Redis command. End
the complete output with one newline.

## Implementation

Add one Cobra command file under `cmd/` and one focused test file. Use only the
Go standard library and the existing Cobra dependency. Keep generation in a
pure function that returns the complete output string; the command writes it to
`cmd.OutOrStdout()`.

Do not add database clients, configuration files, output-file flags, or direct
execution behavior.

## Errors

Return an error and write no generated output when:

- `--uid` is missing or does not exactly match the supported syntax.
- Either suffix bound is outside the required three-digit representation.
- The start is greater than the end.
- A generated UID does not fit a signed 64-bit MySQL `BIGINT`.
- `--token-prefix` is empty or contains whitespace.

## Verification

Focused tests must cover:

- Inclusive generation for `'3181538941{001,010}'`.
- Zero padding in UID, username, email, Redis key, and Redis value.
- The default and explicitly supplied token prefixes.
- Exactly two empty lines between the MySQL and Redis blocks.
- Invalid template, reversed range, overflowing UID, and invalid token prefix.

Run `go test ./cmd/...` and `git diff --check` before completion.

## Deliberate Limits

The command generates one fixed MindCraft test-user shape. It does not accept a
source row, customize individual columns, check existing IDs or emails, or
execute the generated commands. Add those capabilities only if a real reuse
case requires them.
