# Security Policy

## Supported Versions

Currently, only the latest version of this project is supported.

## Reporting a Vulnerability

If you discover a security vulnerability, please do not open a public issue.

Instead, please send an email to **security@example.com** with the following information:

- Description of the vulnerability
- Steps to reproduce the issue
- Impact of the vulnerability
- Suggested fix (if known)

### What to Expect

- We will acknowledge receipt of your report within 48 hours
- We will provide a detailed response within 7 days
- We will work with you to understand and resolve the issue
- You will be credited when the fix is released (unless you prefer to remain anonymous)

### Security Update Process

1. Validation and triage of the vulnerability
2. Development of a fix in a private branch
3. Coordinated release of security advisory and patch
4. Public disclosure after the fix is deployed

## Security Features

This project implements the following security measures:

### Docker Sandbox Container
- Non-root user execution (nobody, UID 65534)
- All capabilities dropped (CapDrop: ALL)
- Network isolation (NetworkMode: none)
- Resource limits (CPU/memory)
- Tmpfs for /tmp
- Read-only root filesystem
- No-new-privileges flag
- PIDs limit (100) to prevent fork bombs
- Memory swap disabled
- Input validation (code size limit: 1MB)
- Seccomp profile (network syscalls blocked)

## Responsible Disclosure

We believe in responsible disclosure. Please:
- Give us reasonable time to investigate and fix the issue
- Do not exploit the vulnerability for any malicious purpose
- Do not disclose the vulnerability publicly until a fix is released

Thank you for helping keep this project secure!
