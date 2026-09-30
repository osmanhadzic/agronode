
---

### `.github/ISSUE_TEMPLATE/security_issue.md`

```markdown
---
name: "🔒 Security Issue"
about: "Report a potential security vulnerability"
title: "[SECURITY] "
labels: "security"
assignees: ""
---

## ⚠️ Do Not Include Sensitive Information

Do not publish:

- Private keys
- Passwords
- Access tokens
- Device certificates
- Production credentials
- Customer data

For potentially exploitable vulnerabilities, use the repository's private security reporting mechanism instead of publicly documenting the exploit.

## Summary

<!-- Briefly describe the potential security issue without sensitive details. -->

## Affected Component

- [ ] Device Registration
- [ ] Device Authentication
- [ ] MQTT / VerneMQ
- [ ] TLS / mTLS
- [ ] API
- [ ] Web Application
- [ ] Database
- [ ] Infrastructure
- [ ] Other

## Impact

<!-- Describe the potential impact without providing exploit instructions. -->

## Affected Versions

<!-- Which versions are affected? -->

## Suggested Mitigation

<!-- Optional. -->

## Additional Context

<!-- Non-sensitive information only. -->
