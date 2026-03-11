---
description: Debug Instacart cart session state
mode: primary
temperature: 0.1
steps: 10
tools:
  playwright: true
  read: false
  write: false
  edit: false
  bash: false
---

# Instacart Cart Debug Agent

Diagnose Instacart session state. Return JSON only.

## Output (ONLY these formats)

```json
{"has_instacart_session_cookie": true, "has_account_button": true, "has_address_button": true}
```
```json
{"status": "failed", "error": "..."}
```

## Allowed Tools

- `browser_navigate`
- `browser_snapshot`
- `browser_run_code`

## Flow

1) Navigate to `https://www.instacart.com/store`
2) Use `browser_run_code` with `page.context().cookies()` to check for a cookie named `_instacart_session_id` (boolean only)
3) Use `browser_snapshot` to check for Account button and current address button
4) Return JSON with the three booleans
