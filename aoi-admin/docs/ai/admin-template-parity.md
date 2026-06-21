# 外部后台 Parity Notes

> 历史记录：本文是 2026-06 的长任务证据和切片记录，不是当前架构事实来源。当前代码边界以 `docs/ai/project-map.md`、`docs/architecture/layers.md` 和对应模块文档为准。

Verified on 2026-06-11 from the public demo and upstream documentation.
Sources:

- 外部后台演示站
- 外部后台项目服务端文档
- 外部后台项目服务端源码

## Persistent Task Book

This file is the durable handoff point for incremental 外部后台 parity.
Before implementing a parity slice, update or append a short entry here with:

1. target slice and current status;
2. whether the public demo must be researched again;
3. visual evidence needed before/after the change;
4. backend/source reference needed before coding;
5. docs, example config, and validation commands to update or run.

Captcha rule: if the public demo login asks for a captcha, stop and ask the
user for the visible captcha value. Do not store local project credentials in
this repository document.

Status legend:

- `[done]`: implemented and documented in this repository.
- `[doing]`: actively being implemented in the current parity slice.
- `[audit]`: present locally but needs 外部后台 comparison or polish before calling
  it equivalent.
- `[next]`: preferred next implementation slice.
- `[todo]`: not implemented or intentionally deferred.
- `[defer]`: possible later work that needs a stronger product decision.

### Current Snapshot

Last local audit: 2026-06-13. External demo/source evidence below remains from
2026-06-12 unless a slice states a newer date.

- Demo access: `外部后台演示站` opened directly into an
  authenticated admin dashboard in the in-app Browser. A separate clean
  external browser session hit login captcha during breakpoint upload
  interaction research, so captcha-protected demo upload completion remains
  blocked until the user provides the visible captcha value.
- Visual evidence saved for this session:
  `tmp/ai/demo-dashboard-2026-06-12.png`,
  `tmp/ai/demo-api-token-2026-06-12.png`,
  `tmp/ai/demo-api-token-dialog-2026-06-12.png`,
  `tmp/ai/demo-version-2026-06-12.png`,
  `tmp/ai/demo-version-export-drawer-2026-06-12.png`,
  `tmp/ai/demo-version-import-drawer-2026-06-12.png`,
  `tmp/ai/demo-upload-2026-06-12.png`,
  `tmp/ai/demo-upload-import-url-2026-06-12.png`,
  `tmp/ai/version-desktop-1440x900-final.png`,
  `tmp/ai/version-export-dialog-1440x900-final.png`,
  `tmp/ai/version-mobile-390x844-final.png`,
  `tmp/ai/api-token-desktop-1440x900-final.png`,
  `tmp/ai/api-token-issue-dialog-1440x900-final.png`,
  `tmp/ai/api-token-mobile-390x844-final.png`, and
  `tmp/ai/api-token-mobile-dialog-390x844-final.png`,
  `tmp/ai/media-desktop-1440x900-final.png`,
  `tmp/ai/media-import-url-1440x900-final.png`,
  `tmp/ai/media-mobile-390x844-final.png`, and
  `tmp/ai/media-mobile-390x844-viewport.png`,
  `tmp/ai/demo-breakpoint-2026-06-12.png`, and
  `tmp/ai/demo-breakpoint-diagnose-2026-06-12.png`,
  `tmp/ai/media-resumable-desktop-1440x900-final.png`,
  `tmp/ai/media-resumable-selected-1440x900-final.png`,
  `tmp/ai/media-resumable-complete-1440x900-final.png`,
  `tmp/ai/media-resumable-mobile-390x844-final.png`,
  `tmp/ai/media-resumable-mobile-390x844-viewport.png`,
  `tmp/ai/customer-desktop-1440.png`,
  `tmp/ai/customer-drawer-1440.png`,
  `tmp/ai/customer-mobile-390.png`,
  `tmp/ai/users-desktop-1440-final.png`, and
  `tmp/ai/users-mobile-390-final.png`.
- Customer/resource example research: 外部后台 route
  `外部后台演示站/#/layout/example/customer` was opened in the
  in-app Browser on 2026-06-12. Screenshot capture timed out in the demo tab,
  so the current evidence is DOM/source based: warning bar, `新增` action, empty
  table with columns `接入日期`、`姓名`、`电话`、`接入人ID`、`操作`, pagination, and
  a `客户` drawer containing `客户名` and `客户电话`.
- Demo visual readout: fixed white shell, left menu, thin dividers, compact
  operational cards, dense tables, small action buttons, minimal decorative
  effects, and visible diagonal demo watermark.
- Local parity stance: continue mapping 外部后台 responsibilities into this repo's
  existing `internal/modules/*/{model,repository,service,handler}`,
  `internal/transport/http`, `internal/app/initapp`, `internal/config`, and
  `pkg` boundaries instead of renaming the backend to 外部后台的 folders.
- Template/code generator stance checked on 2026-06-13: current local capability
  is limited to offline `pkg/sqlgen` and `pkg/yaml2go` libraries plus the
  sqlgen-backed `db` CLI demo workflow. There is no committed admin route,
  backend API, menu entry, permission, runtime config, or static WebUI workflow
  for template config/code generator/form generator/export template features.

### Route And Feature Board

| Status | External area | Local target | Research gate | Notes |
| --- | --- | --- | --- | --- |
| [done] | Admin shell, left menu, visited tabs, dense table styling | `web/admin/app` layout and shared CSS | Visual before/after required for every future UI slice | Keep low-noise 管理后台式 management style. |
| [done] | Dashboard baseline | `/admin` | Re-check demo dashboard before major dashboard redesign | Local dashboard is service/IAM-focused, not plugin-market focused. |
| [done] | Menu management | `/admin/menus`, `/api/v1/system/menus` | External page optional unless changing interactions | Server-driven menu catalog is the local source of truth. |
| [done] | API management and sync | `/admin/apis`, `/api/v1/system/apis` | External page optional unless changing access/filter UI | Includes access-mode summary and permission sync. |
| [done] | Role authorization matrix | `/admin/roles` | Re-check 外部后台 role page before editing permission UX | Local implementation maps to Casbin domain RBAC. |
| [done] | User management | `/admin/users` and IAM APIs | External route/source checked on 2026-06-12 | First pass added list filters, pagination, and compact role/status operations without exposing source wording. |
| [done] | Organization/tenant management | `/admin/organizations` | Research closest external permission pages before changing | Added organization filters, pagination, and compact management UI. |
| [done] | Session/security/MFA pages | `/admin/sessions`, `/admin/security` | Visual and workflow check required before UI changes | Added organization-scoped session paging, security summary, and MFA polish. |
| [done] | Dictionary management | `/admin/dictionaries`, system dictionary APIs | External page optional unless changing item editing UX | Persisted dictionaries and items are implemented. |
| [done] | Operation history | `/admin/operation-records` | Re-check demo before adding advanced filters/export | Persisted protected request records are implemented. |
| [done] | Parameter management | `/admin/parameters` | External page optional unless adding batch/import/export | Persisted parameter CRUD is implemented. |
| [done] | System config read/write | `/admin/system`, `/api/v1/system/config` | Research required before widening writable fields or reload behavior | Current API returns a masked runtime snapshot and supports controlled persistence for approved fields. |
| [done] | Server status | `/admin/server-info`, `/api/v1/system/server-info` | Visual check required if charts are added | Uses local runtime and host metrics. |
| [done] | Login log | `/admin/login-logs` | External page already observed as menu/tab with limited content | Local page uses IAM `auth.login` audit records. |
| [done] | Error log | `/admin/error-logs` | External page observed as unassigned route for admin account | Local page uses `system_operation_records` status filters. |
| [done] | Login captcha | `/api/v1/auth/captcha`, `/admin/login` | Demo login captcha must be checked if login screen changes | Optional config: `auth.login_captcha_enabled`. |
| [done] | API Token | `/admin/api-tokens`, `/api/v1/orgs/:orgId/api-tokens` | External page and external source inspected on 2026-06-12 | Local opaque token implementation with one-time plaintext display and hash-only storage. |
| [done] | Version management | `/admin/versions`, `/api/v1/system/versions` | External page and external source inspected on 2026-06-12 | 外部后台 feature is a configuration release package for selected menus, APIs, and dictionaries; local import safely persists dictionaries and stores menu/API package records. |
| [done] | Media library upload/download | `/admin/media`, `/api/v1/system/media/*` | External page and external source inspected on 2026-06-12 | 外部后台 feature is left-category media management with upload, URL import, keyword filter, preview, rename, download, and delete. |
| [done] | Breakpoint upload | `/admin/media/resumable`, `/api/v1/system/media/assets/resumable/*` | External page and external source inspected on 2026-06-12 | Local implementation uses resumable sessions, SHA-256 chunk/full-file verification, Storage-backed chunk cleanup, and media asset completion. |
| [done] | Customer/resource example | `/admin/customers`, `/api/v1/demo/customers` | Demo visual/source checked on 2026-06-12 | Protected resource CRUD example using local IAM principal scope. |
| [audit] | Template config/code generator/form generator/export template | `pkg/sqlgen`, `pkg/yaml2go`, plus explicit product spec | External workflow/source and security review required before implementation | 2026-06-13 audit confirmed no runtime admin/API surface; keep implementation deferred until product scope is approved. |
| [defer] | AI workflow, MCP Tools, Skills, AI page drawing | Separate AI tooling boundary | Research required before any local work | Keep AI artifacts under `docs/ai` or `tools/ai`; do not mix into app packages. |
| [defer] | Plugin market/install/package/mail plugin/announcements | Existing `plugins` module plus product spec | Research required | Avoid remote marketplace/install behavior without an explicit requirement. |

### Next Slice Protocol

Preferred next slice: Template config/code generator/form generator/export
template external workflow/source research and implementation decision, unless
the user redirects to another external admin area.

Before implementation:

- Research the selected external admin workflow visually and capture screenshots for the
  main list/form, important dialogs, empty state, error state, and completion
  state where the demo exposes them.
- Inspect external admin server and web source for request shape, service
  behavior, cleanup rules, validation, permissions, and failure responses.
- Inspect the closest local module, config, migration status, and frontend
  route before deciding whether to extend an existing boundary or add a new
  one.
- Write the slice plan here before editing code.

Implementation guardrails:

- Treat persistence, permissions, file writes, imports, exports, and background
  cleanup as higher-risk surfaces that need explicit service-level rules.
- Use append-only migrations for new shared tables or columns.
- Keep file-system writes below configured roots, sanitize user-supplied names,
  generate server-side object keys, and never trust uploaded or imported paths.
- Add IAM permissions, server-driven menu entries, API catalog entries, and
  admin pages together when a feature becomes user-visible.
- Update developer, maintainer, user, and beginner-facing docs when behavior is
  exposed.
- Update `configs/*.example.yaml` and `.env.example` when config knobs or
  operational notes are introduced.
- Do not display upstream/source names or "reference demo" wording in product
  UI, user-facing docs, example config comments, or release docs.

Validation floor:

- Run focused Go tests for changed backend packages.
- Run `pnpm typecheck` for changed `web/admin` TypeScript/Vue code.
- For visible UI work, visually inspect desktop `1440x900` and mobile
  `390x844` routes with Browser and record results here or in the final note.

### Active Slice: Template And Generator Audit

Status: `[audit]` started and completed on 2026-06-13. Implementation remains
deferred until product scope, security rules, and external workflow evidence are
available.

Research completed before implementation:

- Read the repo-level and `web/admin` agent rules, AI workspace index, config
  docs, WebUI docs, design rules, build docs, and known gaps before editing.
- Inspected the current config and static chain:
  `internal/config.WebUIConfig -> internal/app/initapp.NewHTTPServer ->
  internal/transport/http.WebUIDeps -> pkg/web.MountStaticSPA`, with Nuxt
  `NUXT_APP_BASE_URL` and Go `webui.mount_path` remaining the path-alignment
  contract.
- Inspected the frontend call chain: admin pages call `useAdminApi()`, shared
  endpoints belong in `web/admin/app/config/admin-api.ts`, and Nuxt runtime
  config only exposes `adminBaseURL`, `apiBaseURL`, `apiMock`, and
  `showDemoTodo`.
- Inspected the style system: admin layout, density, table, filter, status, and
  local scroll values live under `--aoi-admin-*` and Aoi semantic tokens in
  `web/admin/app/assets/css/main.css` and `tokens.css`.
- Inspected local generation libraries: `pkg/sqlgen` supports offline SQL and
  DDL-to-Go generation, while `pkg/yaml2go` is a pure conversion library that
  deliberately does not write files. The committed `db` CLI uses sqlgen only
  for database/Demo schema preview and Demo Todo operations.

Local implementation plan:

- Do not add a generator page, backend API, permissions, menu entries,
  migrations, or build output in this slice.
- Persist the audit result in this task book, configuration docs, and known
  gaps so the next slice has a clear entry point.
- Keep `configs/config.example.yaml`, `.env.example`, and production examples
  unchanged because this audit introduces no runtime configuration.
- Require a future slice to define a product spec before code work: generated
  artifact types, write targets, overwrite policy, field mapping rules,
  validation strategy, permission model, audit logging, export/download format,
  cleanup/rollback behavior, and Browser verification route.

Implementation completed:

- Recorded that the template/code generator/form generator/export template area
  is not a current runtime capability.
- Documented that `pkg/sqlgen` and `pkg/yaml2go` are offline/development tools,
  not user-visible admin workflows.
- Added the follow-up gap to the backlog instead of introducing placeholder
  runtime config or UI.
- Added `docs/ai/generator-product-spec.md` as the product and security gate
  that must be answered before any generator implementation.

Validation completed:

- Documentation-only change. No Go, Vue, Nuxt config, static asset, or CSS
  runtime files were changed.
- Browser visual checks were not required because there was no visible UI
  change. They remain required before any future generator admin surface or
  backend response shape becomes user-visible.

### Active Slice: Template And Generator Product Spec Gate

Status: `[audit]` started and completed on 2026-06-13. Implementation remains
deferred; this slice only records the gated specification that future code work
must satisfy.

Research completed before implementation:

- Re-read the local project structure, config docs, WebUI/static hosting docs,
  frontend API chain, build notes, style rules, generator package READMEs, and
  known gaps.
- Confirmed again that no committed backend route, frontend page, permission,
  menu, runtime config, Nuxt runtime config, or build output exists for the
  generator area.

Local implementation plan:

- Add a durable product/security gate under `docs/ai`.
- Link the gate from the AI index, this task book, configuration notes, and
  known gaps.
- Do not update Go code, Nuxt code, example config, static assets, migrations,
  generated output, or `configs/config.yaml`.
- Keep future configuration names as candidates only; do not imply that runtime
  config exists today.

Implementation completed:

- Added `docs/ai/generator-product-spec.md`.
- Updated `docs/ai/README.md`, `docs/environment/configuration.md`, and
  `docs/backlog/known-gaps.md` so the next slice starts from the same gate.

Risk notes:

- The new document is intentionally marked `draft/gated` to avoid implying that
  the feature is shipped.
- Candidate config names are examples for a future config review, not accepted
  keys.

### Active Slice: Session Security Management

Status: `[done]` started and completed on 2026-06-13.

Research completed before implementation:

- External login-log page was not accessible to the current demo account; it
  returned the no-route/no-permission state. No captcha was shown.
- External source inspection showed the login-log management shape: inline
  filters for username and status, a dense table with IP, success/failure
  status, details, browser/device, login time, row delete, batch delete, and
  pagination.
- External profile page was also not accessible to the current demo account.
  Source inspection showed a profile card, editable basic information, and a
  password-change dialog. Local IAM does not yet expose profile edit or
  password-change endpoints, so this slice will not add those account mutation
  surfaces.

Local implementation plan:

- Preserve local JWT/refresh-session and TOTP design. Do not add password
  change, avatar, phone, or email editing in this slice because those need
  dedicated validation and notification flows.
- Extend `GET /api/v1/orgs/:orgId/sessions` from a bare array to a paginated
  object. Keep no-query calls scoped to the current user for compatibility;
  add `scope=org` for organization-wide session management.
- Filter sessions by current organization in service code even when `userId`
  is supplied, closing the cross-organization listing gap.
- Support `keyword`, `userId`, `ipAddress`, `status`, `scope`, `orderKey`,
  `desc`, `page`, and `pageSize` query fields. Status values are `active`,
  `revoked`, and `expired`.
- Update `/admin/sessions` into a compact management surface with context
  warning, filters, status badges, pagination, and revoke actions. Keep revoke
  disabled for revoked or expired sessions.
- Polish `/admin/security` so it presents account security state, MFA setup,
  and a clear link to session management without adding unsupported profile
  mutations.
- Update API, OpenAPI, IAM, onboarding, maintenance, and overview docs.

Validation plan:

- Add focused IAM service and HTTP router tests for session filtering,
  organization scoping, pagination, and query parsing.
- Run `go test ./internal/modules/iam/... ./internal/transport/http -count=1 -mod=readonly`.
- Run `pnpm --dir web/admin typecheck`.
- Parse `docs/api/openapi.yaml`.
- Visually inspect `/admin/sessions` and `/admin/security` at `1440x900` and
  `390x844`, checking filter wrapping, table scroll containment, MFA fields,
  and text overlap.

Implementation completed:

- Extended `GET /api/v1/orgs/:orgId/sessions` to return `SessionPage` with
  `scope`, `keyword`, `userId`, `ipAddress`, `status`, `orderKey`, `desc`,
  `page`, and `pageSize` query support.
- Kept no-query calls scoped to the current user for compatibility, and added
  `scope=org` for organization-wide session management.
- Fixed the service boundary so session listing always filters by the
  principal's current organization, including when `userId` is supplied.
- Added IAM service and HTTP router tests for session filtering, pagination,
  organization scoping, and query parsing.
- Reworked `/admin/sessions` into a compact management page with filters,
  status badges, pagination, current-session indication, and revoke actions.
- Reworked `/admin/security` into an account security summary with current
  organization, current session, token expiry, MFA state, session-management
  entry, and MFA setup/verification flow.
- Updated API docs, OpenAPI, IAM module docs, onboarding, maintenance, and
  overview notes.

Validation completed:

- `go test ./internal/modules/iam/... ./internal/transport/http -count=1 -mod=readonly`
- `pnpm --dir web/admin typecheck`
- `docs/api/openapi.yaml` parsed successfully with PyYAML.
- Local visual evidence saved:
  `tmp/ai/sessions-desktop-1440-final.png`,
  `tmp/ai/sessions-mobile-390-final.png`,
  `tmp/ai/security-desktop-1440-final.png`, and
  `tmp/ai/security-mobile-390-final.png`.
- Visual notes: desktop session table keeps action visible without page-level
  horizontal overflow; long session/user IDs are truncated instead of
  polluting adjacent cells; mobile filters stack vertically and table overflow
  remains inside the table area; security summary cards wrap long token/session
  values without overlap.

### Active Slice: Organization Management Filters And Pagination

Status: `[done]` started and completed on 2026-06-12.

Research completed before implementation:

- The external admin navigation exposes permission-oriented management pages
  such as role, menu, API, user, dictionary, operation history, parameters,
  token, login log, version, and error log, but no dedicated organization or
  tenant page under the current demo account.
- Direct access to the closest role-management route returned a no-permission
  state for the current demo account. No captcha was shown.
- Local conclusion: keep this repo's organization boundary as the source of
  truth, and borrow only the management-page structure already established in
  adjacent slices: warning/context strip, compact filters, dense table,
  explicit pagination, and a side panel for create/update actions.

Local implementation plan:

- Extend `GET /api/v1/orgs` from a bare array to a paginated object with
  `keyword`, `code`, `name`, `status`, `orderKey`, `desc`, `page`, and
  `pageSize` query support. Default no-query calls return page 1.
- Keep organization creation and current-organization update rules unchanged:
  creating an organization adds the current user as owner; updating is only
  allowed for the token-bound current organization.
- Update `/admin/organizations` into a compact management surface with filters,
  page-size control, paginated table, current-organization edit panel, and
  create panel. Do not add new columns or schema fields in this slice.
- Update API docs, OpenAPI, IAM module docs, onboarding and maintenance notes,
  and keep example configuration wording free of source/reference names.

Validation plan:

- Add focused IAM service and HTTP router tests for organization list
  filtering, sorting, pagination, and query parsing.
- Run `go test ./internal/modules/iam/... ./internal/transport/http -count=1 -mod=readonly`.
- Run `pnpm --dir web/admin typecheck`.
- Parse `docs/api/openapi.yaml`.
- Visually inspect `/admin/organizations` at `1440x900` and `390x844`, with
  special attention to filter wrapping, table scroll containment, and text
  overlap.

Implementation completed:

- Extended `GET /api/v1/orgs` to return `OrganizationPage` with `keyword`,
  `code`, `name`, `status`, `orderKey`, `desc`, `page`, and `pageSize` query
  support.
- Added IAM service and HTTP router tests for organization filtering,
  sorting, pagination, and query parsing.
- Reworked `/admin/organizations` into a compact management page with warning
  context, filters, page-size control, paginated table, current organization
  edit panel, and create panel.
- Trimmed nonessential table columns after visual review so desktop users can
  see the switch action without page-level horizontal scrolling.
- Updated API docs, OpenAPI, IAM module docs, onboarding, maintenance, and
  overview notes.

Validation completed:

- `go test ./internal/modules/iam/... ./internal/transport/http -count=1 -mod=readonly`
- `pnpm --dir web/admin typecheck`
- `docs/api/openapi.yaml` parsed successfully with PyYAML.
- Local visual evidence saved:
  `tmp/ai/organizations-desktop-1440-final.png` and
  `tmp/ai/organizations-mobile-390-final.png`.
- Visual notes: desktop has no page-level horizontal overflow and the action
  column is visible; mobile keeps filters single-column and confines table
  overflow to the table container; rendered body text contains no source or
  reference wording.

### Active Slice: User Management Filters And Pagination

Status: `[done]` started and completed on 2026-06-12.

Research completed before implementation:

- Demo route attempted:
  `外部后台演示站/#/layout/superAdmin/user`. The current public
  demo session returned the 外部后台 no-route/no-permission page, so this role cannot
  visually inspect the live user table without a different demo permission set.
  No captcha was shown.
- Upstream primary source checked at commit
  `78b3bd0c67a2e5cbf469d7e6c5ceceb6fe5546a4`:
  `web/src/view/superAdmin/user/user.vue`,
  `web/src/api/user.js`,
  `server/model/system/sys_user.go`,
  `server/model/system/request/sys_user.go`,
  `server/service/system/sys_user.go`,
  `server/router/system/sys_user.go`, and system menu/API/casbin seeds.
- External user-management shape: warning bar, inline filters for username,
  nickname, phone, and email; a paginated user table sorted by ID descending;
  avatar, username, nickname, phone, email, role cascader, enable switch, and
  row actions for delete, edit, and reset password; add/edit user drawer; reset
  password dialog.

Local implementation plan:

- Preserve this repo's organization-scoped IAM design. Do not add phone/avatar
  columns in this slice because local `iam_users` does not currently model
  those fields and adding schema solely for visual parity would be noisy.
- Extend organization user listing with 管理后台式 server-side filters and
  pagination: keyword, username, display name, email, role code, membership
  status, page, and page size. Keep the endpoint path
  `/api/v1/orgs/:orgId/users` and make no-query calls return page 1.
- Update `/admin/users` into a compact 管理后台式 list surface with warning bar,
  search/reset controls, page-size selector, paginated table, role/status quick
  actions, and invitation side panel. Keep invitation as the local equivalent
  of "新增用户" until a deliberate direct-user-create slice is planned.
- Update API docs, OpenAPI, IAM module docs, onboarding/maintenance notes, and
  AI task status after validation.

Validation plan:

- Focused Go tests for IAM service and HTTP router user-list query handling.
- `go test ./internal/modules/iam/... ./internal/transport/http -count=1 -mod=readonly`
- `pnpm --dir web/admin typecheck`
- Local visual checks on `/admin/users` at `1440x900` and `390x844`, including
  populated table, filter controls, and invitation panel.

Implementation completed:

- Extended `GET /api/v1/orgs/:orgId/users` to return `OrganizationUserPage`
  with `keyword`, `username`, `displayName`/`nickName`, `email`, `roleCode`,
  `status`, `orderKey`, `desc`, `page`, and `pageSize` query support.
- Added IAM service and HTTP router tests for filtered/paginated user lists.
- Updated `/admin/users` into a compact organization-user management surface
  with filters, pagination, role/status quick actions, and invitation panels.
- Updated API composable/types and the API Token issue dialog to consume
  paginated user metadata via `items`.
- Removed upstream/source wording from product UI and user-facing docs.

Validation completed:

- `go test ./internal/modules/iam/... ./internal/transport/http -count=1 -mod=readonly`
- `pnpm --dir web/admin typecheck`
- `docs/api/openapi.yaml` parsed successfully with PyYAML.
- API smoke using a local test account verified `/api/v1/orgs/{orgId}/users`
  returns a persisted page object and filters `keyword=admin`.
- Local visual evidence saved:
  `tmp/ai/users-desktop-1440-final.png` and
  `tmp/ai/users-mobile-390-final.png`.
- Visual notes: desktop filters no longer overlap the invitation panel; mobile
  keeps filters vertical and the dense member table inside a local horizontal
  scroll area; rendered body text contains no upstream/source wording.

### Active Slice: Customer Resource Example

Status: `[done]` started and completed on 2026-06-12.

Research completed before implementation:

- Demo route: `外部后台演示站/#/layout/example/customer`.
- Demo visual readout: a compact white resource example page, warning bar,
  primary `新增` action, table columns for access date, customer name, phone,
  access-user ID and operations, empty state `暂无数据`, and bottom pagination.
  The create/edit surface is a right-side `客户` drawer with `客户名` and
  `客户电话` fields plus cancel/confirm actions.
- Browser screenshot capture timed out on the public demo tab, so no 外部后台
  customer screenshot is saved yet. DOM inspection and external source were
  used for this slice; local screenshots are still required after
  implementation.
- Upstream primary source checked at commit
  `78b3bd0c67a2e5cbf469d7e6c5ceceb6fe5546a4`:
  `web/src/view/example/customer/customer.vue`,
  `web/src/api/customer.js`,
  `server/model/example/exa_customer.go`,
  `server/model/example/response/exa_customer.go`,
  `server/api/v1/example/exa_customer.go`,
  `server/service/example/exa_customer.go`,
  `server/router/example/exa_customer.go`,
  `server/source/system/menu.go`, `server/source/system/api.go`, and
  `server/source/system/casbin.go`.

Local implementation plan:

- Keep the feature in the Demo module because it is an example CRUD resource,
  but register it behind IAM authentication and `customer:*` permissions to
  mirror 外部后台的 private router plus Casbin responsibility split.
- Add a `demo_customers` model/table with customer name, phone, owner user ID,
  owner username, owner role code, organization ID, timestamps, and soft delete.
  It is intentionally separate from System media/version tables because it is a
  teaching example rather than operational configuration.
- Apply local resource visibility as: users list customers from their current
  organization where the owner role matches their active role, or records they
  created themselves. This maps 外部后台的 role data-authority example into the
  current IAM principal shape without adding 外部后台的 full authority association
  model.
- Expose protected APIs under `/api/v1/demo/customers` for list, create, detail,
  update, and delete. Use `customer:read`, `customer:create`,
  `customer:update`, and `customer:delete`; add them to built-in permissions,
  API catalog mapping, and the server-driven menu.
- Add `/admin/customers` with a compact 管理后台式 warning bar, `新增` action,
  table, pagination, drawer form, and mobile responsive layout. Use existing
  admin components and avoid decorative backgrounds or nested card clutter.
- Update developer, secondary-development, maintainer, user, beginner, API,
  OpenAPI, Demo module, configuration, and AI handoff docs. Example config
  comments should say the Demo switch now controls both public Todo and the
  protected Customer resource example.

Validation plan:

- Run focused Go tests for Demo service, app db/init transport, and HTTP router,
  then `go test ./... -count=1 -mod=readonly`.
- Run `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`.
- Run `pnpm --dir web/admin typecheck`.
- Use Browser or Playwright visual checks on `/admin/customers` at `1440x900`
  and `390x844`, including list and create-drawer states.

Implementation completed:

- Added `demo_customers` model, repository, service, handler, schema bootstrap,
  IAM permissions, HTTP route permission mapping, API catalog mapping, and
  server-driven menu entry.
- Added `/admin/customers` with a 管理后台式 warning bar, compact search tools,
  table pagination, create/edit drawer, delete flow, and mobile horizontal table
  handling.
- Updated Demo, API, OpenAPI, configuration, onboarding, overview, maintenance,
  extension, and AI handoff docs plus example config comments.

Validation completed:

- `go test ./internal/modules/demo/... ./internal/app/dbapp ./internal/app/initapp ./internal/app ./internal/transport/http -count=1 -mod=readonly`
- `go test ./... -count=1 -mod=readonly`
- `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`
- `go build -mod=readonly -o ./tmp/go-scaffold-server.exe ./cmd/aoi`
- `pnpm --dir web/admin typecheck`
- `docs/api/openapi.yaml` parsed successfully with PyYAML.
- API smoke using a local test account verified login, customer detail, list,
  update, and UTF-8 Chinese keyword filtering. Windows PowerShell string bodies
  must be sent as UTF-8 bytes for Chinese request payloads.
- Local visual evidence saved:
  `tmp/ai/customer-desktop-1440.png`,
  `tmp/ai/customer-drawer-1440.png`, and
  `tmp/ai/customer-mobile-390.png`.
- Visual notes: desktop list and right drawer are readable with no body-level
  horizontal overflow; mobile keeps the dense customer table inside a horizontal
  scroll area and avoids global layout pollution.

### Active Slice: Breakpoint Upload

Status: `[done]` started and completed on 2026-06-12.

Research completed before implementation:

- Demo route: `外部后台演示站/#/layout/example/breakpoint`.
- Visual evidence: `tmp/ai/demo-breakpoint-2026-06-12.png`.
- External page shape: a compact white panel titled "大文件上传", one "选择文件"
  file picker button, one "上传文件" primary action, a selected-file row with
  filename, percentage and a thin progress bar, plus a note that the test
  version writes chunks and merged files into server-side breakpoint/file
  folders. The demo limit shown in the UI is 5 MB.
- External clean-browser login required a captcha during upload interaction
  research. Per the captcha rule, the user was asked for the visible captcha
  value; until provided, upload progress/completion screenshots remain pending.
- Upstream primary source checked at commit
  `78b3bd0c67a2e5cbf469d7e6c5ceceb6fe5546a4`:
  `web/src/view/example/breakpoint/breakpoint.vue`,
  `web/src/api/breakpoint.js`,
  `server/api/v1/example/exa_breakpoint_continue.go`,
  `server/service/example/exa_breakpoint_continue.go`,
  `server/model/example/exa_breakpoint_continue.go`,
  `server/router/example/exa_file_upload_and_download.go`,
  `server/utils/breakpoint_continue.go`,
  `server/source/system/menu.go`,
  `server/source/system/api.go`, and `server/source/system/casbin.go`.

Local implementation plan:

- Keep the feature under System media because resumable upload creates the same
  operational media assets as ordinary upload. Add a dedicated
  `/admin/media/resumable` page and a server-driven demo menu item next to the
  media library.
- Add append-only tables `system_media_upload_sessions` and
  `system_media_upload_chunks`. Sessions store file hash, original filename,
  category, size, chunk size/count, status (`active`, `completed`, `aborted`,
  `expired`), final asset ID, uploader identity, timestamps and expiry. Chunks
  store session ID, chunk index, byte size, chunk hash, storage key and upload
  timestamp.
- Expose protected APIs under `/api/v1/system/media/assets/resumable`: check or
  create session, upload one chunk, complete/merge, and abort/cleanup. Reuse
  `media:upload` for check/chunk/complete/abort; listing and final asset access
  continue to use existing media permissions.
- Use SHA-256 for full-file and chunk hashes, while keeping request field names
  close enough to 外部后台的 `fileHash`, `chunkHash`, `chunkIndex`, `chunkTotal`,
  and `fileName` responsibility split.
- Reuse the existing `MediaMaxBytes` limit and `MediaPrefix`; default chunk
  size is 1 MB and the server rejects empty files, files over the max, invalid
  chunk indexes/counts, mismatched chunk hashes, and sessions that are complete,
  aborted or expired.
- Keep all temporary objects below a server-generated
  `media/chunks/<session-id>/` prefix, never trust client file paths, and merge
  chunks into a normal `system_media_assets` row with a generated final key.
- Abort and complete should best-effort remove temporary chunk objects. Expired
  sessions remain visible to the backend as non-uploadable; a later maintenance
  job can hard-delete stale chunk data if needed.
- Build the frontend page with the 管理后台式 compact panel, choose/upload
  buttons, selected-file row, progress bar, existing-chunk resume state, storage
  unavailable warning, and links back to the media library.
- Update developer, secondary-development, maintainer, user, beginner, API,
  OpenAPI, system module, storage/config and AI handoff docs.

Validation plan:

- Run focused Go tests for `internal/modules/system/service`,
  `internal/modules/system/handler`, `internal/transport/http`, and `pkg/web`
  where touched, then `go test ./... -count=1 -mod=readonly`.
- Run `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`.
- Run `pnpm typecheck` from `web/admin`.
- Use Browser visual checks on `/admin/media/resumable` at `1440x900` and
  `390x844`, including storage-disabled and selected-file states. If the user
  provides the 外部后台 captcha, also capture 外部后台 upload progress/completion
  evidence for comparison.

Implementation completed:

- Added append-only migration
  `internal/migrations/20260612000700_create_system_media_resumable_uploads.sql`
  for `system_media_upload_sessions` and `system_media_upload_chunks`.
- Added System media models, repository methods, service methods, handlers, and
  protected routes for resumable `check`, `chunks`, `complete`, and `abort`.
- Reused `media:upload`, `MediaMaxBytes`, `MediaPrefix`, and the injected
  `pkg/storage` object store. Temporary chunks are written below
  `media/chunks/<session-id>/`, and complete/abort best-effort remove chunk
  objects plus chunk rows.
- Implemented `/admin/media/resumable` with 管理后台式 choose/upload/reset/abort
  actions, selected-file row, progress bar, resume state, storage unavailable
  warning, and completion link back to the media library.
- Moved the media library route from `pages/media.vue` to
  `pages/media/index.vue` after visual validation found `/media/resumable`
  was otherwise rendering the parent media page.
- Updated API, OpenAPI, module, configuration, extension, maintenance,
  onboarding, overview and AI docs, plus `.env.example` and
  `configs/config.example.yaml`.

Validation completed:

- `go test ./internal/modules/system/service -count=1 -mod=readonly`
- `go test ./internal/modules/system/... ./internal/transport/http ./pkg/web -count=1 -mod=readonly`
- `go test ./... -count=1 -mod=readonly`
- `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`
- `go build -mod=readonly -o ./tmp/go-scaffold-server.exe ./cmd/aoi`
- `pnpm --dir web/admin typecheck`
- `python -c "import pathlib, yaml; yaml.safe_load(pathlib.Path('docs/api/openapi.yaml').read_text(encoding='utf-8')); print('openapi yaml parsed')"`
- Browser desktop visual check on `/admin/media/resumable` at `1440x900`.
- Edge/Playwright upload flow using a generated 1.5 MB `tmp/ai` file: selected
  state recognized two chunks, upload completed, and the final asset appeared
  in the page result.
- Visual evidence:
  `tmp/ai/media-resumable-desktop-1440x900-final.png`,
  `tmp/ai/media-resumable-selected-1440x900-final.png`,
  `tmp/ai/media-resumable-complete-1440x900-final.png`,
  `tmp/ai/media-resumable-mobile-390x844-final.png`, and
  `tmp/ai/media-resumable-mobile-390x844-viewport.png`.

### Active Slice: Media Library Upload/Download

Status: `[done]` started and completed on 2026-06-12.

Research completed before implementation:

- Demo route: `外部后台演示站/#/layout/example/upload`.
- Visual evidence:
  `tmp/ai/demo-upload-2026-06-12.png` and
  `tmp/ai/demo-upload-import-url-2026-06-12.png`.
- External page shape: left category tree with `全部分类`; top warning bar; action
  buttons for normal upload, crop upload, QR upload, compressed upload, and
  URL import; keyword filter; table columns for preview, date, file name/remark,
  link, tag, and row operations; pagination at the lower right.
- Interaction readout: file name is editable from the table row; URL import
  accepts newline-separated `文件名|链接` or bare URL entries in a prompt.
- Upstream primary source checked:
  `web/src/view/example/upload/upload.vue`,
  `web/src/api/fileUploadAndDownload.js`,
  `web/src/api/attachmentCategory.js`,
  `server/model/example/exa_file_upload_download.go`,
  `server/model/example/exa_attachment_category.go`,
  `server/model/example/request/exa_file_upload_and_downloads.go`,
  `server/model/example/response/exa_file_upload_download.go`,
  `server/api/v1/example/exa_file_upload_download.go`,
  `server/api/v1/example/exa_attachment_category.go`,
  `server/service/example/exa_file_upload_download.go`,
  `server/service/example/exa_attachment_category.go`,
  `server/router/example/exa_file_upload_and_download.go`, and
  `server/router/example/exa_attachment_category.go` from
  `外部后台项目源码`.

Local implementation plan:

- Put the user-visible management surface in the System module because local
  media records are operational admin assets backed by `pkg/storage`, not a
  throwaway demo table.
- Add append-only tables `system_media_categories` and `system_media_assets`.
  Categories store ID, parent ID, name, sort, timestamps, and soft delete.
  Assets store ID, category ID, display name, original filename, storage key,
  URL/path, MIME type, extension/tag, byte size, source (`upload` or `url`),
  external flag, uploader identity, timestamps, and soft delete.
- Inject optional `pkg/storage.Storage` into the System service. When storage is
  disabled, list/imported URL records can still be visible from DB, but binary
  upload/download/delete of local objects must return a clear storage
  unavailable error.
- Expose protected APIs under `/api/v1/system/media`: category tree, create or
  update category, delete category, asset list, upload, URL import, rename,
  download, and delete.
- Add IAM permissions `media:read`, `media:upload`, `media:import`,
  `media:update`, `media:download`, and `media:delete`, then wire them through
  API catalog, role permission matrix, and server-driven menus.
- Build `/admin/media` with the 管理后台式 layout: left category panel, compact
  action/filter row, preview table/list, URL import dialog, rename dialog, and
  desktop/mobile responsive behavior. Start with normal upload and URL import;
  crop/QR/compress buttons may be represented as deferred actions only if they
  are not functionally implemented in this slice.
- Keep storage safety explicit: server-generated keys under a `media/` prefix,
  sanitized display names, size limit, MIME sniffing, no path traversal, no URL
  fetching during import, and best-effort object deletion when DB rows are
  deleted.
- Update developer, maintainer, user, beginner, API, OpenAPI, system module,
  storage/config, and AI handoff docs. Example config should document storage
  enablement and the media prefix/limit if new knobs are added.

Validation plan:

- Run focused Go tests for `internal/modules/system` and
  `internal/transport/http`, then `go test ./... -count=1 -mod=readonly`.
- Run `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`.
- Run `pnpm typecheck` from `web/admin`.
- Use Browser visual checks on `/admin/media` at `1440x900` and `390x844`,
  including URL import and normal empty/non-empty states where possible.

Implementation completed:

- Added append-only migration
  `internal/migrations/20260612000600_create_system_media.sql`, the media
  category and asset models, repository methods, service workflow, handlers,
  and protected HTTP routes for category tree, asset list, normal upload, URL
  import, rename, download, and delete.
- Reused `pkg/storage` through dependency injection. With storage disabled,
  media list and external URL imports still work, while local binary
  upload/download/delete report a clear unavailable state.
- Added IAM permissions `media:read`, `media:upload`, `media:import`,
  `media:update`, `media:download`, and `media:delete`, then wired them into
  the route catalog, server-driven menu, and role permission matrix.
- Added `/admin/media` with the 管理后台式 left category panel, action/filter
  row, warning bars, table/card resource list, URL import dialog, rename
  dialog, and responsive desktop/mobile behavior.
- Kept storage safety explicit: server-generated object keys under `media/`,
  sanitized display names, upload size limits, MIME sniffing, no trusted client
  paths, no remote URL fetching during import, and best-effort object deletion.
- Fixed a visual/runtime issue found during QA: nullable media API item lists
  are normalized before Vue renders `flatMap`, preventing a blank update cycle
  when an unavailable or empty storage-backed catalog returns `items: null`.
- Updated API, OpenAPI, system module, README, onboarding, maintenance,
  extension, environment, overview, IAM, AI handoff, and example config notes.

Validation completed:

- `go test ./internal/modules/system/service ./internal/transport/http ./internal/modules/system/handler ./pkg/web -count=1 -mod=readonly`
- `go test ./... -count=1 -mod=readonly`
- `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`
- `pnpm typecheck` from `web/admin`
- Playwright/Edge visual checks after Browser tooling was unavailable:
  `tmp/ai/media-desktop-1440x900-final.png`,
  `tmp/ai/media-import-url-1440x900-final.png`,
  `tmp/ai/media-mobile-390x844-final.png`, and
  `tmp/ai/media-mobile-390x844-viewport.png`.

### Active Slice: Version Management

Status: `[done]` started and completed on 2026-06-12.

Research completed before implementation:

- Demo route: `外部后台演示站/#/layout/admin/sysVersion`.
- Visual evidence:
  `tmp/ai/demo-version-2026-06-12.png`,
  `tmp/ai/demo-version-export-drawer-2026-06-12.png`, and
  `tmp/ai/demo-version-import-drawer-2026-06-12.png`.
- External page shape: list page filters by created date, version name, and version
  code; primary actions are create release package and import version package;
  row actions are view, download package, and delete. The create drawer collects
  version name/code/description and lets the user select menus, APIs, and
  dictionaries. The import drawer accepts a JSON package, previews its menus,
  APIs, and dictionaries, then imports it.
- Upstream primary source checked:
  `web/src/view/systemTools/version/version.vue`,
  `web/src/api/version.js`,
  `server/model/system/sys_version.go`,
  `server/model/system/request/sys_version.go`,
  `server/model/system/response/sys_version.go`,
  `server/api/v1/system/sys_version.go`,
  `server/service/system/sys_version.go`, and
  `server/router/system/sys_version.go` from
  `外部后台项目源码`.

Local implementation plan:

- Keep the 外部后台 responsibility split, but name the local concept explicitly as a
  system release package. It snapshots menus, API routes, and dictionaries into
  a versioned JSON payload instead of representing the running binary version or
  migration version.
- Add an append-only `system_versions` migration and model with ID, version
  name, version code, description, JSON payload, package counts, source
  (`export` or `import`), creator/importer, created/updated/deleted timestamps.
- Expose protected APIs under `/api/v1/system/versions` for list, detail,
  source catalog, export, import, download, single delete, and batch delete.
- Add IAM permissions `version:read`, `version:create`, `version:import`,
  `version:download`, and `version:delete`, then wire them through API catalog,
  server-driven menus, and the role permission matrix.
- Build `/admin/versions` with the same low-noise table/filter/drawer workflow
  observed in the external reference: filters, selection, create package, import package,
  detail preview, JSON download, and batch delete.
- Preserve local architecture on import: dictionaries and dictionary items can
  be created idempotently when missing; menus and API routes are code/router
  owned in this scaffold, so imported menu/API entries are recorded in the
  package and reported as skipped until those catalogs become safely mutable.
- Update developer, maintainer, user, beginner, API, OpenAPI, and AI handoff
  docs. No example config change is expected unless a new configuration knob is
  introduced.

Validation plan:

- Run focused Go tests for `internal/modules/system` and
  `internal/transport/http`, then `go test ./... -count=1 -mod=readonly`.
- Run `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`.
- Run `pnpm typecheck` from `web/admin`.
- Use Browser visual checks on `/admin/versions` at `1440x900` and `390x844`,
  including create/import/detail workflows where possible.

Implementation completed:

- Added append-only migration
  `internal/migrations/20260612000500_create_system_versions.sql`, the
  `system_versions` model, repository methods, service workflow, handlers, and
  protected HTTP routes for source catalog, list, detail, export, import,
  download, single delete, and batch delete.
- Added IAM permissions `version:read`, `version:create`, `version:import`,
  `version:download`, and `version:delete`, then wired them into the route
  catalog, role permission matrix, and server-driven system menu.
- Added `/admin/versions` with 管理后台式 filters, table selection, create
  release package workflow, import JSON workflow, detail/JSON download support,
  and mobile responsive layout.
- Preserved local architecture on import: dictionaries and items are created
  idempotently when missing; menus and API routes remain code/router-owned and
  are reported as skipped while still stored in the package JSON.
- Updated API, OpenAPI, system module, README, onboarding, maintenance,
  extension, environment, overview, AI handoff, and example config notes.
- Fixed visual pollution found during QA: the version page's page-size filter,
  export dialog width/scroll behavior, and mobile filter actions now render
  without clipping or navigation overlap.

Validation completed:

- `go test ./... -count=1 -mod=readonly`
- `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`
- `pnpm typecheck` from `web/admin`
- Browser/CDP visual checks:
  `tmp/ai/version-desktop-1440x900-final.png`,
  `tmp/ai/version-export-dialog-1440x900-final.png`, and
  `tmp/ai/version-mobile-390x844-final.png`.

### Active Slice: API Token Management

Status: `[done]` started and completed on 2026-06-12.

Research completed before implementation:

- Demo route: API Token page under Super Admin. Screenshots:
  `tmp/ai/demo-api-token-2026-06-12.png` and
  `tmp/ai/demo-api-token-dialog-2026-06-12.png`.
- External page shape: filters by user ID and status; primary action is issue;
  table columns are ID, user, role ID, status, expires at, remark, operation;
  issue drawer asks for user, role, validity period, and remark; success dialog
  shows the token once; operation area includes curl example and invalidate.
- Upstream primary source checked:
  `web/src/view/systemTools/apiToken/index.vue`,
  `web/src/api/sysApiToken.js`,
  `server/model/system/sys_api_token.go`,
  `server/model/system/request/sys_api_token.go`,
  `server/api/v1/system/sys_api_token.go`,
  `server/service/system/sys_api_token.go`, and
  `server/router/system/sys_api_token.go` from
  `外部后台项目源码`.

Local implementation plan:

- Keep the 外部后台 management workflow shape, but implement local API tokens as
  opaque secrets with a display prefix and SHA-256 hash. Do not store the
  plaintext token or copy 外部后台的 raw JWT persistence.
- Put the backend in IAM because the token authenticates callers and belongs to
  users, organizations, and role/permission scope. Keep route registration in
  `internal/transport/http`.
- Add an append-only migration for `iam_api_tokens` with organization, user,
  optional role, name, remark, prefix, token hash, status, expiration, last-used
  metadata, and created/revoked audit columns.
- Expose protected organization-scoped APIs under
  `/api/v1/orgs/:orgId/api-tokens` for list, create, and revoke. List supports
  user, status, and pagination. Create returns the plaintext token only in the
  create response.
- Add permissions `api_token:read`, `api_token:create`, and
  `api_token:revoke`, then surface them in menu, API catalog, and role matrix.
- Add `/admin/api-tokens` using existing Aoi admin components and the same
  low-noise table/filter/drawer pattern as other system pages.
- Update docs for developer, maintainer, user, and beginner readers. Update
  example config only if implementation introduces a new config knob.

Implementation completed:

- Added `iam_api_tokens` through append-only migration
  `internal/migrations/20260612000400_create_iam_api_tokens.sql`.
- Added IAM repository, service, handler, auth fallback, and permission checks
  for opaque API tokens. Plaintext tokens are returned only by create responses;
  persisted records store prefix plus hash.
- Added organization-scoped protected APIs for list, create, and revoke under
  `/api/v1/orgs/:orgId/api-tokens`.
- Added route catalog entries, IAM permissions, server-driven menu entry, and
  `/admin/api-tokens`.
- Fixed the shared `AoiSelect` Material select value/display synchronization
  issue found during visual inspection.
- Updated API, IAM, onboarding, maintenance, extension, environment, OpenAPI,
  and AI handoff documentation. Example config comments now document that the
  refresh-token pepper also protects API token hashes.

Validation completed:

- `go test ./internal/modules/iam/... ./internal/transport/http/... ./internal/modules/system/... -count=1 -mod=readonly`
- `go test ./... -count=1 -mod=readonly`
- `go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi`
- `pnpm typecheck` from `web/admin`
- Browser visual checks:
  `tmp/ai/api-token-desktop-1440x900-final.png`,
  `tmp/ai/api-token-issue-dialog-1440x900-final.png`,
  `tmp/ai/api-token-mobile-390x844-final.png`, and
  `tmp/ai/api-token-mobile-dialog-390x844-final.png`.

Follow-up note: no background cleanup job was added for expired tokens. List
and authentication paths compute expired state at read/auth time, which is
sufficient for the current management workflow.

## Visual Reference

- Shell: fixed left menu, top toolbar, visited tabs, dense white work surface.
- Navigation: menu groups expand in place; active item uses a strong blue block.
- Data pages: filters stay at the top, actions sit above the table, and tables
  favor compact row height with operation controls on the right.
- Dashboard: summary cards, chart/table regions, quick-entry panels, and notice
  lists are arranged as operational widgets rather than marketing cards.
- Styling: the demo keeps backgrounds mostly solid white, uses thin borders, and
  avoids blurred or translucent surfaces inside core management workflows.
- Visual pollution to avoid while replacing 外部后台 incrementally: front-site
  background images, colorful navigation gradients, translucent glass panels,
  large marketing gradients, decorative blur, high-opacity watermark patterns,
  and low-contrast table/header text.

## Visual Review Rule

For future parity work, use screenshot or browser-based visual inspection before
and after implementation whenever a frontend change affects the UI, or a backend
change affects an admin workflow that users can see. Record the route, viewport,
and remaining risk in the handoff or final note.

Required minimum viewports for Aoi Admin visual work:

- Desktop: `1440x900`.
- Mobile: `390x844`.

When the local admin account requires MFA, record the blocked route and continue
with visual checks that do not require authenticated state until the code is
available.

## Backend Reference

外部后台's server is organized around `api/v1`, `config`, `core`,
`global`, `initialize`, `middleware`, `model`, `model/request`,
`model/response`, `router`, `service`, `source`, and `utils`.

This repository should map that pattern into its existing boundaries instead of
renaming the backend wholesale:

- `api/v1` maps to `internal/modules/*/handler` plus
  `internal/transport/http`.
- `router` maps to `internal/transport/http` route registration.
- `service` maps to `internal/modules/*/service`.
- `model`, `request`, and `response` map to module-local model/DTO packages or
  `types/result` for shared envelopes.
- `initialize` maps to `internal/app/initapp`.
- `core` maps to `internal/app` plus reusable `pkg` infrastructure.
- `config` maps to `internal/config`.
- `middleware` maps to `internal/middleware`.
- `utils` maps to reusable packages under `pkg`.

Do not rename this repository to match 外部后台的 folder names wholesale. The parity
target is the responsibility split: router catalog, API handler, service domain
rules, repository persistence, typed request/response shapes, initialization,
middleware, and reusable utilities.

外部后台的 router initialization separates public routes from private routes guarded
by JWT and Casbin. In this scaffold, the equivalent information is expressed in
the API catalog as `access=public|authenticated|permission` while the concrete
middleware remains in `internal/transport/http` and `internal/middleware`.

## Incremental Replacement Order

1. Stabilize the admin shell and table-page visual system.
2. Keep IAM pages aligned with the backend's existing organization, role, user,
   session, security, and audit APIs.
3. Add missing backend management modules only when the Go server exposes real
   models and routes.
4. Preserve the current dependency rule: modules depend on reusable `pkg`
   infrastructure, while `pkg` does not import application modules.
5. Avoid copying 外部后台's code generator, plugin market, or generated
   CRUD surface until this backend has an explicit product requirement for them.

## Audit And Deferred Records

- 2026-06-13: Template config/code generator/form generator/export template
  audit confirmed the repo only has offline `pkg/sqlgen` and `pkg/yaml2go`
  tool libraries plus the sqlgen-backed `db` CLI demo workflow. No runtime
  admin route, API endpoint, permission, migration, WebUI page, build output,
  YAML setting, env var, or Nuxt runtime config was added. The next phase must
  start with a product specification and security review covering artifact
  types, write targets, overwrite policy, field mapping, permissions, audit
  logging, export/download format, cleanup, rollback, and Browser-visible
  verification routes.

## Implemented Parity Slices

- 2026-06-11: Admin visual cleanup plus static icon bundling.
- 2026-06-11: Server-driven admin menu groups at `/api/v1/system/menus`.
- 2026-06-11: HTTP API catalog at `/api/v1/system/apis`, mapped from the
  current router table.
- 2026-06-11: 管理后台式 API sync action at `/api/v1/system/apis/sync`, backed by
  `system_apis` when the migration has been applied and safely downgraded to
  live in-memory catalog refresh when the table is not available yet.
- 2026-06-11: API permission dictionary sync at
  `/api/v1/system/apis/permissions/sync`, deriving IAM permission records from
  registered backend routes so the role authorization page can bind them.
- 2026-06-11: Role authorization page changed from a flat permission list to a
  grouped permission matrix with object filters, keyword search, per-group bulk
  selection, and API-management handoff.
- 2026-06-12: Menu management catalog page added at `/admin/menus`, showing the
  server-driven menu groups, route paths, permission bindings, mobile entries,
  icons, and order values that back the admin shell.
- 2026-06-12: Dictionary management slice added with persisted
  `system_dictionaries` and `system_dictionary_items`, CRUD HTTP APIs, IAM
  permissions, role-matrix grouping, a server-driven menu entry, and the
  `/admin/dictionaries` management page.
- 2026-06-12: Operation history slice added after visually inspecting 外部后台的
  `操作历史` page: protected API requests are recorded into
  `system_operation_records`, surfaced through `/api/v1/system/operation-records`,
  wired into IAM permissions and server-driven menus, and managed from
  `/admin/operation-records` with 管理后台式 filters, selection, table layout, and
  pagination.
- 2026-06-12: Parameter management slice added after checking 外部后台的
  `参数管理` / `sys_params` model and service: persisted `system_parameters`
  records expose name, key, value, description, created timestamps, list filters,
  single and batch delete, key lookup, IAM permissions, server-driven menus, and
  the `/admin/parameters` management page.
- 2026-06-12: System configuration slice added after checking 外部后台的
  `系统配置` page and `/system/getSystemConfig` route: this scaffold initially
  exposed a permission-protected `/api/v1/system/config` masked runtime
  snapshot, wired `config:read` into IAM/menu/API catalogs, and added the
  `/admin/system` grouped configuration page. The current implementation has
  since grown controlled persistence for approved fields; widening writable
  fields or service reload behavior remains a higher-risk parity slice.
- 2026-06-12: Server status slice added after checking 外部后台的
  `/system/getServerInfo` service shape: this scaffold now exposes
  `/api/v1/system/server-info` with `server:read`, returns gopsutil-backed
  host CPU/RAM/disk metrics plus Go runtime, memory, GC, OS, uptime, and build
  metadata, wires the server-driven menu and role permission matrix, and adds
  `/admin/server-info`.
- 2026-06-12: Admin visual pollution hardening after visual comparison with
  外部后台 dashboard and menu-management pages: the admin runtime now clears legacy
  Aoi background/colorful-nav variables, and the admin CSS baseline uses a
  restrained 管理后台式 palette with solid panels, thin borders, low shadows,
  denser tables, muted login branding, semantic API method badges, isolated
  admin surface tokens, and desktop/mobile visual checks.
- 2026-06-12: 外部后台 `source`/`initialize` parity slice: the System module can
  seed default dictionaries and parameters during startup through
  `system.seed_defaults_on_start`. The seed is idempotent, skips unavailable
  tables, and never overwrites existing user-edited parameter values.
- 2026-06-12: Login log parity slice after inspecting 外部后台
  `#/layout/admin/loginLog`: the demo currently exposes the menu item and tab
  but keeps dashboard content in the work surface, so this scaffold implements a
  usable `/admin/login-logs` page backed by IAM `auth.login` audit records and
  adds the server-driven menu entry under Security Audit.
- 2026-06-12: Error log parity slice after inspecting 外部后台
  `#/layout/admin/errorLog`: the public demo currently renders the unassigned
  route/permission page for the admin account, so this scaffold implements a
  usable `/admin/error-logs` page over `system_operation_records`. The backend
  keeps the existing operation-record table and adds optional `statusClass`
  filtering (`4xx`, `5xx`, or `error`) to `/api/v1/system/operation-records`;
  exact `status` filters still take priority when supplied.
- 2026-06-12: API catalog access-mode parity slice based on 外部后台的 public vs
  JWT/Casbin-protected router groups: route catalog entries now expose
  `access` as `public`, `authenticated`, or `permission`, and the API management
  page can summarize and filter by that access mode without changing the
  append-only `system_apis` schema.
- 2026-06-12: Login captcha parity slice based on the external login screen:
  IAM now exposes public `GET /api/v1/auth/captcha`, validates optional
  `captchaId`/`captchaCode` during login when `auth.login_captcha_enabled=true`,
  keeps short-lived challenges in service memory, and renders the admin login
  captcha row only when the backend reports it enabled.
- 2026-06-12: API Token management parity slice after inspecting the external reference
  page and upstream `sys_api_token` source: this scaffold now stores
  organization-scoped opaque API tokens by hash, supports one-time plaintext
  display on issue, list/status filtering, revoke, API-token Bearer auth with
  role permission scope, server-driven menu/API catalog entries, and the
  `/admin/api-tokens` management page. During visual QA, the shared
  `AoiSelect` component was fixed so Material select values render reliably in
  desktop and mobile dialogs.
- 2026-06-12: Version management parity slice after inspecting the external reference
  `sysVersion` page and upstream `sys_version` source: this scaffold now stores
  versioned system release packages for selected menus, APIs, and dictionaries,
  exposes `/api/v1/system/versions` source/export/import/download/delete
  workflows, wires `version:*` IAM permissions and menu/API catalogs, adds the
  `/admin/versions` management page, and documents the safe local import rule
  where dictionaries are persisted while menu/API entries remain code-owned.
- 2026-06-12: Media library upload/download parity slice after inspecting the
  外部样板 upload page and upstream file/category source: this scaffold now
  stores media categories and assets, supports normal storage-backed upload,
  external URL import, keyword filtering, rename, download/open, delete, IAM
  `media:*` permissions, server-driven menus/API catalog entries, and the
  `/admin/media` management page. Breakpoint/chunk upload remains the preferred
  next slice because it needs a separate storage protocol and cleanup design.
