# Security Checks

Run these commands from the repository root. Keep generated reports under
`tmp/ai` unless an audit artifact is explicitly requested.

## Quick Pass

```powershell
go test ./... -count=1 -mod=readonly
golangci-lint run --config tools/ai/golangci.yml ./...
govulncheck ./...
gosec ./...
osv-scanner scan source .
```

`golangci-lint` uses new-issue mode. For an explicit full historical audit,
temporarily remove `issues.new: true` from `tools/ai/golangci.yml` and expect
the first run to report existing baseline findings.

## Report Outputs

```powershell
New-Item -ItemType Directory -Force tmp/ai
gosec -fmt=json -out=tmp/ai/gosec.json ./...
osv-scanner scan source . --format json > tmp/ai/osv.json
govulncheck -json ./... > tmp/ai/govulncheck.json
```

## Triage Notes

- `govulncheck` findings are strongest when they include reachable call stacks.
- `gosec` findings need human review for intentional examples, test-only code,
  and local development defaults.
- `osv-scanner` may report dependency vulnerabilities that are present but not
  reachable. Pair it with `govulncheck` before choosing an upgrade.
- For authentication, authorization, token, MFA, storage, or migration changes,
  treat a clean scan as supporting evidence, not proof of safety.
