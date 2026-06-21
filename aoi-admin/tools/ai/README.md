# AI Tooling

This directory contains repo-local configuration and runbooks for AI-assisted
development. It is intentionally separate from application code.

## Files

- `golangci.yml`: GolangCI-Lint v2 configuration for local AI and developer
  checks. It runs in new-issue mode so the repo can adopt linting without
  forcing a full historical cleanup first.
- `security-checks.md`: local security scan commands and interpretation notes.

## Output Policy

Write short-lived outputs to `tmp/ai`, for example:

```powershell
New-Item -ItemType Directory -Force tmp/ai
gosec -fmt=json -out=tmp/ai/gosec.json ./...
osv-scanner scan source . --format json > tmp/ai/osv.json
```

Do not commit generated reports unless the user explicitly asks for an audit
artifact.
