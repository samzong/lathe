# Security Hardening

This document tracks repository-level security controls for Lathe. Some controls live in versioned files, while others must be enabled in GitHub repository settings.

## Versioned Controls

- CI: `.github/workflows/ci.yml`
- Release automation: `.github/workflows/release.yml`
- CodeQL scanning: `.github/workflows/codeql.yml`
- OpenSSF Scorecard: `.github/workflows/scorecard.yml`
- Dependabot version updates: `.github/dependabot.yml`
- Security policy: `SECURITY.md`

## GitHub Settings

Verified on 2026-05-18:

- Dependabot vulnerability alerts: enabled through the GitHub API.
- Dependabot automated security fixes: enabled through the GitHub API.

Still required after the workflows land on `main`:

- Protect `main`.
- Require pull requests before merging.
- Require at least one approving review.
- Dismiss stale approvals when new commits are pushed.
- Require status checks to pass before merging.
- Add the stable CI, CodeQL, and Scorecard check names after the first successful run on `main`.
- Disable force pushes and deletions on `main`.
- Enable private vulnerability reporting if it is not already available under GitHub Security Advisories.
- Enable secret scanning and push protection when available for the repository plan.

## Release Security

Current release automation publishes checksums through GoReleaser. The next hardening steps are:

- Sign release artifacts or provenance.
- Document checksum verification in release notes.
- Consider SLSA provenance once release binaries are stable.
