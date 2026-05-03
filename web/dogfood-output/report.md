# Dogfood QA Report

**Target:** http://localhost:5173/
**Date:** May 03, 2026
**Scope:** Full Korus Web Console UX evaluation — all pages, navigation, forms, i18n
**Tester:** Hermes Agent (automated exploratory QA)

---

## Executive Summary

| Severity | Count |
|----------|-------|
| 🔴 Critical | 0 |
| 🟠 High | 2 |
| 🟡 Medium | 3 |
| 🔵 Low | 2 |
| **Total** | **7** |

**Overall Assessment:** The Korus Web Console has a clean, professional UI with good navigation structure and i18n support. Two high-severity form validation issues and several medium/low UX polish items need attention before production use.

---

## Issues

### Issue #1: Form validation not working — edit form

| Field | Value |
|-------|-------|
| **Severity** | 🟠 High |
| **Category** | Functional |
| **URL** | `/tenants/:tenantId/workspaces/:workspaceId` (edit mode) |

**Description:**
When editing a workspace, clearing the required "显示名称" (Display Name) field and clicking "保存更改" (Save Changes) does not trigger any validation feedback. The form silently ignores the submission attempt — no error messages, no red borders, no visual indication of what's wrong.

**Steps to Reproduce:**
1. Navigate to a workspace detail page
2. Click "编辑" (Edit) button
3. Clear the "显示名称" field
4. Click "保存更改" (Save Changes)

**Expected Behavior:**
Form should display a validation error message (e.g., "显示名称为必填项") and highlight the invalid field with a red border.

**Actual Behavior:**
Nothing happens — the form stays in edit mode with no feedback to the user.

**Screenshot:**
MEDIA:/home/windosx/dev/korus/web/dogfood-output/screenshots/05-validation-no-feedback.png

---

### Issue #2: Cancel button not working — edit form

| Field | Value |
|-------|-------|
| **Severity** | 🟠 High |
| **Category** | Functional |
| **URL** | `/tenants/:tenantId/workspaces/:workspaceId` (edit mode) |

**Description:**
The "取消" (Cancel) button on the workspace edit form does not work. Clicking it does not exit edit mode or return to the detail view. Users must use the breadcrumb navigation to escape the edit form.

**Steps to Reproduce:**
1. Navigate to a workspace detail page
2. Click "编辑" (Edit) button
3. Click "取消" (Cancel) button

**Expected Behavior:**
The form should close and return to the workspace detail view.

**Actual Behavior:**
Nothing happens — the form remains in edit mode. Clicking Cancel multiple times has no effect.

**Screenshot:**
MEDIA:/home/windosx/dev/korus/web/dogfood-output/screenshots/04-workspace-edit.png

---

### Issue #3: No validation feedback on workspace create form

| Field | Value |
|-------|-------|
| **Severity** | 🟡 Medium |
| **Category** | Functional |
| **URL** | `/tenants/:tenantId/workspaces/new` |

**Description:**
The workspace create form has required fields (ID, 标识, 显示名称) but no validation feedback when submitting with empty fields. The form silently fails without informing the user which fields are required.

**Steps to Reproduce:**
1. Navigate to workspace create page
2. Leave all required fields empty
3. Click "创建工作区" (Create Workspace)

**Expected Behavior:**
Form should highlight empty required fields and display error messages.

**Actual Behavior:**
Nothing happens — the form stays on the same page with no feedback.

**Screenshot:**
MEDIA:/home/windosx/dev/korus/web/dogfood-output/screenshots/06-workspace-create.png

---

### Issue #4: Missing required field indicators

| Field | Value |
|-------|-------|
| **Severity** | 🟡 Medium |
| **Category** | UX |
| **URL** | `/tenants/:tenantId/workspaces/new` |

**Description:**
The workspace create form has required fields (ID, 标识, 显示名称) but does not visually indicate which fields are required. There are no asterisks, "required" labels, or other visual markers to guide users.

**Steps to Reproduce:**
1. Navigate to workspace create page
2. Observe the form fields

**Expected Behavior:**
Required fields should have visual indicators (e.g., asterisks *, "required" labels, or different styling).

**Actual Behavior:**
All fields appear visually optional — users have no way to know which fields are required before submitting.

**Screenshot:**
MEDIA:/home/windosx/dev/korus/web/dogfood-output/screenshots/06-workspace-create.png

---

### Issue #5: Raw ISO 8601 timestamps in Runs table

| Field | Value |
|-------|-------|
| **Severity** | 🟡 Medium |
| **Category** | UX |
| **URL** | `/tenants/:tenantId/runs` |

**Description:**
The Runs table displays timestamps in raw ISO 8601 UTC format (e.g., `2026-04-29T09:10:00Z`), which is not human-friendly for quick scanning.

**Steps to Reproduce:**
1. Navigate to the Runs page
2. Observe the "开始时间" (Start Time) column

**Expected Behavior:**
Timestamps should be displayed in a localized, human-readable format (e.g., "2026年4月29日 09:10" or "Apr 29, 2026 09:10 UTC").

**Actual Behavior:**
Raw ISO 8601 format is displayed, which requires mental parsing.

**Screenshot:**
MEDIA:/home/windosx/dev/korus/web/dogfood-output/screenshots/09-runs-list.png

---

### Issue #6: Runtime column text wrapping

| Field | Value |
|-------|-------|
| **Severity** | 🔵 Low |
| **Category** | Visual |
| **URL** | `/tenants/:tenantId/runs` |

**Description:**
The "运行时" (Runtime) column in the Runs table is too narrow, causing the text "eino adk" to wrap to two lines.

**Steps to Reproduce:**
1. Navigate to the Runs page
2. Observe the "运行时" column

**Expected Behavior:**
Column should be wide enough to display "eino adk" on a single line.

**Actual Behavior:**
Text wraps to two lines, reducing readability.

**Screenshot:**
MEDIA:/home/windosx/dev/korus/web/dogfood-output/screenshots/09-runs-list.png

---

### Issue #7: Truncated API URL in Providers table

| Field | Value |
|-------|-------|
| **Severity** | 🔵 Low |
| **Category** | Visual |
| **URL** | `/tenants/:tenantId/providers` |

**Description:**
The first row's API URL in the Providers table is truncated (shows `https://dashscope.aliyun.com/compatible-m...`) with no tooltip or expand option to view the full endpoint.

**Steps to Reproduce:**
1. Navigate to the Providers page
2. Observe the "凭据" (Credentials) column for the first provider

**Expected Behavior:**
Long URLs should either be fully displayed, truncated with a tooltip on hover, or have an expand/copy button.

**Actual Behavior:**
URL is truncated with no way to view the full value.

**Screenshot:**
MEDIA:/home/windosx/dev/korus/web/dogfood-output/screenshots/10-providers-list.png

---

## Issues Summary Table

| # | Title | Severity | Category | URL |
|---|-------|----------|----------|-----|
| 1 | Form validation not working — edit form | 🟠 High | Functional | `/workspaces/:id` (edit) |
| 2 | Cancel button not working — edit form | 🟠 High | Functional | `/workspaces/:id` (edit) |
| 3 | No validation feedback on create form | 🟡 Medium | Functional | `/workspaces/new` |
| 4 | Missing required field indicators | 🟡 Medium | UX | `/workspaces/new` |
| 5 | Raw ISO 8601 timestamps | 🟡 Medium | UX | `/runs` |
| 6 | Runtime column text wrapping | 🔵 Low | Visual | `/runs` |
| 7 | Truncated API URL | 🔵 Low | Visual | `/providers` |

## Testing Coverage

### Pages Tested
- ✅ `/tenants` — Tenant list page
- ✅ `/tenants/:tenantId/workspaces` — Workspace list page
- ✅ `/tenants/:tenantId/workspaces/new` — Workspace create page
- ✅ `/tenants/:tenantId/workspaces/:workspaceId` — Workspace detail page
- ✅ `/tenants/:tenantId/agents` — Agents list page
- ✅ `/tenants/:tenantId/evaluations` — Evaluations list page
- ✅ `/tenants/:tenantId/runs` — Runs list page
- ✅ `/tenants/:tenantId/providers` — Providers list page
- ✅ `/tenants/:tenantId/settings` — Settings page

### Features Tested
- ✅ Tenant switcher dropdown
- ✅ Sidebar navigation
- ✅ Breadcrumb navigation
- ✅ Table rendering and data display
- ✅ Workspace edit form
- ✅ Workspace create form
- ✅ Language switching (zh-CN ↔ en-US)
- ✅ Status badges (Active, Inactive, Published, Draft, Pass, Fail, Success)
- ✅ Provider bindings table
- ✅ Console error checking (zero JS errors across all pages)

### Not Tested / Out of Scope
- ❌ Responsive/mobile layout (tested at 1280x633 viewport only)
- ❌ Keyboard navigation and accessibility (Tab, Enter, screen reader)
- ❌ Delete confirmation dialog
- ❌ Row click navigation on Agents/Evaluations/Runs/Providers tables
- ❌ Pagination behavior (all test data fits on one page)
- ❌ Empty states (all pages have test data)
- ❌ Error states (API failures, network errors)

### Blockers
- None — all pages loaded successfully with zero console errors

---

## Notes

**Strengths:**
- Clean, professional dark sidebar + light content layout
- Consistent navigation patterns across all pages
- High-quality i18n translations (both zh-CN and en-US)
- Zero JavaScript console errors across all pages
- Good use of status badges with color coding
- Secure credential display (secret:// protocol)
- Breadcrumb navigation works correctly

**Recommendations:**
1. **Priority 1:** Fix form validation and Cancel button in workspace edit/create forms
2. **Priority 2:** Add required field indicators to all forms
3. **Priority 3:** Localize timestamp display in Runs table
4. **Priority 4:** Adjust column widths to prevent text wrapping
5. **Priority 5:** Add tooltips or expand functionality for truncated URLs
