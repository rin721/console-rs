# React Admin API Parity Audit

Status: IN_PROGRESS  
Date: 2026-06-19  
Scope: current Go HTTP router routes versus React `web/app` API endpoints, routes, and focused tests.

## Sources Checked

- Backend router: `internal/transport/http/router.go`
- React route registry: `web/app/app/routes.ts`
- React endpoint registry: `web/app/app/lib/api/endpoints.ts`
- React API wrappers: `web/app/app/lib/api/{auth,setup,iam,system,plugins}.ts`
- React E2E contract coverage: `web/app/tests/e2e/smoke.spec.ts`
- Endpoint parity unit coverage: `web/app/app/lib/api/endpoints.test.ts`

## Result

The current Go admin-facing API surface has React endpoint coverage and React route coverage for the public site, setup flow, auth flow, admin IAM, System, probe, media, version, parameter, dictionary, operation-record, server-info, API catalog, API token, session, audit, and plugin query capabilities.

The audit found no new backend-exposed SaaS admin production capability that is completely missing from React after the public settings gap was closed. Core live Go-hosted API/Browser smoke and broader admin mutation, permission, and error-state sampling passed on 2026-06-19. The final completion audit is tracked in `docs/ai/react-frontend-final-audit.md`; remaining work is Docker image verification.

## Route Matrix

| Backend surface | Go routes | Access | React endpoint/client | React UI route | Focused evidence | Status |
| --- | --- | --- | --- | --- | --- | --- |
| Probes | `GET /health`, `GET /ready` | Public | `systemApi.getHealth`, `systemApi.getReady` | `/admin/probes`, dashboard overview | `admin probes route`, `admin probes route renders degraded readiness details`, endpoint unit test | Covered |
| Setup center | `/api/v1/setup/status`, `/schema`, `/configs/:stepKey`, `/configs/:stepKey/test`, `/runs`, `/runs/:id/retry`, `/runs/:id/steps/:stepKey/skip`, `/runs/:id/logs`, `/complete` | Public before setup; setup-token where backend requires it | `setupApi` | `/setup`, `/setup/:step` | setup E2E group, Go setup router integration notes, endpoint unit test | Covered |
| Legacy IAM setup | `GET /api/v1/auth/setup/status`, `POST /api/v1/auth/setup/initial-admin` | Public compatibility | `API_ENDPOINTS.auth.setupStatus`, `API_ENDPOINTS.auth.initialAdminSetup` | No standalone React path; React uses setup center | endpoint unit test, IAM docs note legacy compatibility | Intentionally excluded as legacy compatibility |
| Public auth | `POST /api/v1/auth/signup`, `GET /api/v1/auth/captcha`, `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, password forgot/reset | Public | `authApi.signup`, `captcha`, `login`, client refresh, forgot/reset | `/signup`, `/login`, `/password/forgot`, `/password/reset` | login captcha/MFA E2E, password recovery E2E, endpoint unit test | Covered |
| Invitation accept | `POST /api/v1/invitations/:token/accept` | Public | `authApi.acceptInvitation` | `/invitations/:token` | invitation acceptance E2E, endpoint unit test | Covered |
| Current account | `POST /api/v1/auth/logout`, `/switch-org`, `/mfa/setup`, `/mfa/verify`, `GET /api/v1/me`, `/me/orgs` | Authenticated | `authApi.logout`, `switchOrg`, `setupMFA`, `verifyMFA`, `getMe`, `listMyOrganizations` | `/admin/security`, admin shell org context | admin auth/security E2E, endpoint unit test | Covered |
| Organizations | `GET/POST /api/v1/orgs`, `PATCH /api/v1/orgs/:orgId` | `org:read/create/update` | `iamApi.listOrganizations`, `createOrganization`, `updateOrganization` | `/admin/organizations`, admin shell org context | admin organizations E2E, endpoint unit test | Covered |
| Users and invitations | users list/update/invite plus invitation list/revoke under `/api/v1/orgs/:orgId/*` | `user:read/update/invite` | `iamApi.listUsers`, `updateUser`, `inviteUser`, `listInvitations`, `revokeInvitation` | `/admin/users` | admin users E2E, endpoint unit test | Covered |
| API tokens | `GET/POST/DELETE /api/v1/orgs/:orgId/api-tokens*` | `api_token:read/create/revoke` | `iamApi.listAPITokens`, `createAPIToken`, `revokeAPIToken` | `/admin/api-tokens` | admin API tokens E2E, endpoint unit test | Covered |
| Roles and permissions | roles list/create/update, permissions list | `role:*`, `permission:read` | `iamApi.listRoles`, `createRole`, `updateRole`, `listPermissions` | `/admin/roles` | admin roles E2E, endpoint unit test | Covered |
| Sessions | sessions list/revoke | `session:read/revoke` | `iamApi.listSessions`, `revokeSession` | `/admin/sessions` | admin sessions E2E, endpoint unit test | Covered |
| Audit logs | `GET /api/v1/orgs/:orgId/audit-logs` | `audit:read` | `iamApi.listAuditLogs` | `/admin/audit-logs`, `/admin/login-logs` | audit-log and login-log E2E | Covered |
| Plugin admin query | `GET /api/v1/plugins`, `/:pluginId`, `/:pluginId/health`, `/:pluginId/capabilities` | `plugin:read` | `pluginsApi` | `/admin/plugins` | admin plugins E2E, endpoint unit test | Covered |
| Plugin protocol | `/plugin-api/v1/*` negotiate/register/heartbeat/invoke/etc. | Plugin protocol, not SaaS admin | No React endpoint by design | No React UI | router audit, plugin docs | Intentionally excluded from frontend admin |
| Public system settings | `GET /api/v1/system/public-settings` | Public | `systemApi.getPublicSettings`, `usePublicSettings` | public layout and home JSON-LD | public home E2E, visual check, endpoint unit test, live Go smoke for configured brand fields | Covered |
| System menus | `GET /api/v1/system/menus` | Authenticated menu filter | `systemApi.listMenus` | `/admin/menus` | admin menu catalog E2E, endpoint unit test | Covered |
| Runtime config | `GET/PATCH /api/v1/system/config` | `config:read/update` | `systemApi.getConfig`, `updateConfig` | `/admin/system` | admin system settings E2E | Covered |
| Server info | `GET /api/v1/system/server-info` | `server:read` | `systemApi.getServerInfo` | `/admin/server-info`, dashboard overview | admin server info E2E | Covered |
| API catalog and permission sync | `GET /api/v1/system/apis`, `POST /apis/sync`, `POST /apis/permissions/sync` | `permission:read/sync` | `systemApi.listAPIs`, `syncAPIs`, `syncAPIPermissions` | `/admin/apis` | admin API catalog E2E | Covered |
| Operation records and error logs | `GET/DELETE /api/v1/system/operation-records` | `operation:read/delete` | `systemApi.listOperationRecords`, `deleteOperationRecords` | `/admin/operation-records`, `/admin/error-logs` | operation-record and error-log E2E | Covered |
| Versions | version list/export/import/delete/sources/detail/download | `version:*` | `systemApi` version methods | `/admin/versions` | admin versions E2E, endpoint unit test | Covered |
| Media | category CRUD, asset list/upload/import/update/download/delete, resumable check/chunk/complete/abort | `media:*` | `systemApi` media methods | `/admin/media`, `/admin/media/resumable` | media and resumable E2E, endpoint unit test | Covered |
| Parameters | list/create/bulk delete/key value/detail/update/delete | `parameter:*` | `systemApi` parameter methods | `/admin/parameters` | admin parameters E2E | Covered |
| Dictionaries | dictionary CRUD, dictionary item create/update/delete | `dictionary:*` | `systemApi` dictionary methods | `/admin/dictionaries` | admin dictionaries E2E | Covered |
| Design-system theme settings | No backend persistence contract | Permission-controlled local admin UI only | No backend theme endpoint | `/admin/design-system` | design-system E2E asserts no theme/design API calls | Covered as local-only by backend contract |

## Contract Exclusions

- `/plugin-api/v1/*` remains the remote plugin protocol. It is deliberately outside the React SaaS admin API client because it is not a user-facing admin production workflow.
- `/api/v1/auth/setup/*` remains a legacy IAM setup compatibility surface. React uses `/api/v1/setup/*` for the current first-install wizard.
- Parameter helper routes such as `GET /api/v1/system/parameters/value` are covered through the typed API wrapper, but do not need a dedicated standalone page because the CRUD workflow is the admin-facing surface.
- The design-system page remains local draft/preview/import/export only because the Go backend has no theme persistence, audit, publish, or rollback API.

## Live Smoke Completed

- `go run ./cmd/aoi server` was started with a temporary config under `tmp/ai/live-smoke-fixed` and port `19120`.
- API smoke passed for `/health`, `/ready`, `/api/v1/system/public-settings`, `/api/v1/setup/status`, `/api/v1/setup/runs`, `/api/v1/me`, `/api/v1/orgs`, `/api/v1/system/apis`, and `/api/v1/system/server-info`.
- Browser smoke passed for Go-hosted `/` and authenticated `/admin/server-info` at `1440x900` and `390x844` with no console errors, failed resource requests, or horizontal overflow.
- Broader admin API smoke used a fresh temporary config under `tmp/ai/live-smoke-admin-*` and port `19121`. It verified organization update, API token create/list/revoke, parameter create/read/update/delete, dictionary and item create/update/delete, API catalog sync, permission sync, operation-record listing, unauthenticated 401, invalid request 400, missing resource 404, and restricted-role 403 behavior.

## Remaining Verification

- Run Docker image verification when Docker is available locally or in CI.
- Re-run `docs/ai/react-frontend-final-audit.md` after Docker image verification passes before marking the thread goal complete.
