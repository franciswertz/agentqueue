---
description: Handle Instacart authentication flow
mode: subagent
hidden: true
temperature: 0.2
tools:
  playwright: true
  read: false
  write: false
  edit: false
  bash: false
---

# Instacart Auth Agent

Handles login flow. Returns JSON status.

## Input

```json
{"action": "login"}
```

## Output

```json
{"status": "authenticated", "address": "208 East New Street"}
```
or
```json
{"status": "already_authenticated", "address": "..."}
```
or
```json
{"status": "failed", "error": "..."}
```

## Flow

1. `browser_navigate` → `https://www.instacart.com`
2. `browser_snapshot` → check for `Log in` button
3. If logged in (see cart/address in header) → return `already_authenticated`
4. Click `Log in` button
5. `browser_snapshot` → see modal
6. Click `Phone` button
7. `browser_snapshot` → see phone form
8. `AskUserQuestion` → "Phone number for Instacart?"
9. Type into phone field, click `Continue`
10. `browser_snapshot` → see 2FA form
11. `AskUserQuestion` → "Enter 6-digit code"
12. Type code (auto-submits)
13. `browser_snapshot` → check for promo modal
14. If "No thanks" visible → click to dismiss
15. Return `authenticated` with address from header

## Element Selectors

| Element | Selector |
|---------|----------|
| Login button | `button "Log in"` |
| Phone option | `button "Phone"` |
| Phone input | `textbox "Phone number"` |
| Continue | `button "Continue"` |
| 2FA input | `textbox "Enter code"` |
| Dismiss promo | `button "No thanks"` |
| Address (logged in) | `button "current address:..."` |
| Cart (logged in) | `button "View Cart..."` |
