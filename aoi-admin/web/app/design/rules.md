# Aoi React Design Rules

This file defines the durable React design system and frontend architecture rules for `web/app`.

## Product Surfaces

- Public site: marketing home, about, blog, legal pages, authentication entry points, invitation acceptance, password reset, and public SEO metadata.
- First-install setup: `/setup/*` wizard for language selection, storage, database, cache, initial account, site/system information, tests, recovery, and completion.
- Shared platform console: `/admin` application shell and backend-exposed IAM, System, probe, media, version, plugin query, audit, and design-system/theme configuration capabilities.
- Product-line entries: future public pages, business consoles, feature modules, and domain models that reuse the shared platform foundations instead of duplicating account, organization, permission, configuration, audit, API, i18n, design-system, test, or build logic.
- Shared foundations: design tokens, Aoi React components, i18n, API client, auth state, Markdown rendering, quality tooling, and static build flow.

## Foundation Tokens

- Use two token tiers by default: primitive raw values and semantic usage values.
- CSS variable names use hierarchical hyphen naming, for example `--aoi-color-text-primary`, `--aoi-space-4`, and `--aoi-radius-card`.
- Components must reference semantic tokens. New raw values require a real consumer and a semantic mapping.
- Tailwind v4 utilities must consume the same CSS variables; do not create a parallel Tailwind-only palette.
- Token changes require checking public site and admin surfaces at desktop and mobile breakpoints.

## Visual Language

- Public pages use bright surfaces, restrained accent color, clear type hierarchy, lightweight cards, short CTAs, and media blocks with stable aspect ratios.
- Platform console pages use solid backgrounds, dense but readable information layout, low shadows, clear borders, data tables, filters, action toolbars, and dialogs or drawers for mutations.
- Setup pages use a focused wizard layout, clear progress, trustworthy copy, low distraction, explicit recovery, and visible handling for tests, errors, loading, and completion.
- Avoid one-note palettes. Primary color must be used for selected navigation, primary actions, focus rings, badges, and small highlights only.
- Do not use gradient orbs, bokeh blobs, decorative glass panels, oversized marketing cards, or stock-like images in admin workflows.
- Text must fit its container in both `zh-CN` and `en`; long English words need safe wrapping.

## Theme Settings

- The admin theme settings page manages a runtime-configurable subset of semantic UI tokens only. It must not expose arbitrary source token editing, raw CSS injection, or developer-only token internals.
- The page must remain permission-controlled and backend-contract-driven. Local preview, import/export, and in-memory draft interactions are allowed before backend persistence exists, but save, publish, audit, and rollback must be clearly disabled or labeled non-production until the Go backend exposes those contracts.
- Required controls include theme preview, color and contrast validation, typography scale, spacing, radius, shadow, motion intensity, light/dark mode, restore defaults, draft state, publish confirmation, import/export, and rollback messaging.
- Every configurable value must satisfy WCAG AA contrast where color pairs are involved, responsive layout constraints, and reduced-motion requirements.
- Theme configuration state must not store secrets, auth tokens, private API payloads, raw HTML, script, unsafe URLs, or arbitrary Tailwind classes.
- Theme settings copy, labels, validation errors, table headings, confirmation text, empty/loading/error states, SEO metadata, and a11y labels must live in locale resources.

## Component Layers

- Tokens: colors, typography, spacing, radii, shadows, borders, z-index, motion, breakpoints, and layout constants.
- Primitives: button, icon button, link, input, textarea, select, checkbox, switch, badge, tag, spinner, progress, tooltip, and table cell helpers.
- Patterns: app shell, public header, admin sidebar, admin top bar, setup wizard, step navigation, form field, data table, empty state, error state, loading state, dialog, drawer, dropdown menu, pagination, command/search, markdown prose, and toast.
- Templates: public page template, legal page template, blog index/detail template, auth template, setup page template, admin list page, admin detail page, settings page, and dashboard page.
- Application code may use any layer, but lower layers must not import higher layers.

## Routing

- React Router Framework mode owns route modules in `app/routes`.
- Public routes stay outside `/admin`.
- Setup routes stay under `/setup/*` and must not be nested inside public or platform console shells.
- Shared platform console routes stay under `/admin/*`; route guards must preserve intended redirect targets.
- Future product-line route namespaces must stay separate from `/admin/*` unless they are managing shared platform capability. Product-line code may import shared foundations but must not fork them.
- Route metadata, page titles, SEO fields, nav labels, breadcrumbs, and a11y labels must use i18n keys.
- Do not add a frontend route for a backend capability that lacks API support, permissions, and documented behavior.

## API Integration

- API endpoints live in `app/lib/api/endpoints.ts`.
- API client owns base URL resolution, `X-Locale`, auth headers, token refresh retry, error normalization, abort handling, and downloads.
- TanStack Query query keys must be stable arrays and colocated with API functions or feature modules.
- Mutations must invalidate or update the exact affected query keys.
- Error UI must show normalized error messages and keep trace IDs available when present.

## Forms

- Use React Hook Form and Zod for forms.
- Zod schemas must use i18n keys or translation-aware message builders, not hardcoded runtime copy.
- Every field must expose label, help text when needed, validation message, disabled state, and loading/submitting behavior.
- Authentication forms must avoid leaking whether an account exists unless the backend response explicitly allows it.
- Setup forms must be schema-driven from backend-exposed fields. Sensitive values such as database, cache, storage, and initial account secrets must not be written to local persistent storage.

## i18n And SEO

- Locale resources are source of truth for all user-visible copy.
- SEO title, description, Open Graph title, Open Graph description, canonical path labels, and social alt text must be localized.
- Markdown front matter includes `locale`; blog index must filter by active locale.
- Locale fallback, language preference persistence, and API locale mapping must be centralized in `app/i18n`.
- Hardcoded copy checks must scan `app`, `content`, route config, schemas, navigation config, table column definitions, tests that render UI, and SEO helpers.

## Markdown Prose

- `react-markdown` renders Markdown.
- `remark-gfm` is enabled for tables, task lists, strikethrough, and autolink literals.
- `rehype-sanitize` is mandatory if any raw HTML support is introduced.
- Shiki code highlighting is preferred at build time. The highlighted output must remain sanitized and accessible.
- Prose styles belong to the Aoi Markdown pattern, not page-local CSS.

## Accessibility

- Interactive controls must have visible focus states, keyboard support, accessible names, and semantic roles.
- Dialogs, menus, popovers, and tooltips should use Radix primitives through Aoi wrappers.
- Tables must expose captions or `aria-label`, header scope, loading states, empty states, and keyboard-safe actions.
- Color may not be the only signal for state, selection, validation, or destructive action.
- Reduced motion must disable nonessential movement and skeleton shimmer.

## Verification

- `typecheck` is required for TypeScript, route, API, store, i18n, Markdown, and config changes.
- `lint:i18n` is required for user-visible copy and locale changes.
- Unit or component tests are required for reusable logic, forms, API normalization, Markdown parsing, and Aoi primitives with behavior.
- Playwright or Browser verification is required for visible UI changes at `1440x900` and `390x844`.
- Build verification must confirm the static output contains `index.html` and asset URLs respect the configured base path.
