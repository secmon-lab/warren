---
paths:
  - "frontend/**"
---

# Frontend Rules

## Test Requirements
- **EVERY frontend feature addition or modification MUST include corresponding E2E tests**
- E2E tests are located in `frontend/e2e/tests/` using Playwright
- Page Objects are in `frontend/e2e/pages/` — add or update them as needed
- Use semantic locators (`getByRole`, `getByText`, `data-testid`) instead of CSS class selectors
- Do NOT use `waitForLoadState("networkidle")` — rely on Playwright's auto-waiting
- Do NOT consider a frontend task complete until E2E tests covering the new/changed behavior are written

## Date Format
- ALWAYS use `YYYY/MM/DD` format. NEVER use `MM/DD/YYYY` or locale-dependent formats like `toLocaleDateString()`
  - Use: `date.toISOString().split('T')[0].replace(/-/g, '/')`
  - Do NOT use: `date.toLocaleDateString()`
