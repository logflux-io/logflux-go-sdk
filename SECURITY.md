# Security Policy

## Reporting Security Vulnerabilities

LogFlux takes security seriously. If you discover a security vulnerability in the LogFlux Go SDK, please report it privately to help us maintain the security of the project and its users.

### How to Report

**Please DO NOT create public issues for security vulnerabilities.**

Instead, please report security issues by:

1. **Email**: Send details to security@logflux.io
2. **GitHub Security Advisory**: Use the [private vulnerability reporting feature](https://github.com/logflux-io/logflux-go-sdk/security/advisories/new)

### What to Include

When reporting a security vulnerability, please provide:

- **Description** of the vulnerability
- **Steps to reproduce** the issue
- **Affected versions** of the SDK
- **Potential impact** assessment
- **Suggested fix** (if you have one)
- **Your contact information** for follow-up

## Security Response Process

1. **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
2. **Investigation**: Our team will investigate and assess the vulnerability
3. **Communication**: We will keep you updated on our progress
4. **Resolution**: We will work on a fix and coordinate the disclosure
5. **Credit**: We will credit you in our security advisory (unless you prefer to remain anonymous)

## Supported Versions

Security updates are provided for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

As this is a BETA release, we are actively maintaining security for the current version. Future version support policy will be established upon stable release.

## Security Best Practices

When using the LogFlux Go SDK:

### Authentication
- **Always use authentication** when connecting via TCP
- **Keep shared secrets secure** and rotate them regularly
- **Use Unix sockets** when possible for local agent communication (more secure than TCP)

### Network Security
- **Use TLS** for TCP connections in production environments
- **Restrict network access** to LogFlux agents
- **Monitor agent connections** and watch for unauthorized access attempts

### Data Security
- **Sanitize log data** before sending to prevent information disclosure
- **Avoid logging sensitive data** such as passwords, API keys, or personal information
- **Consider data retention policies** for logged information

### Code Security
- **Keep dependencies updated** by regularly running `go mod tidy && go mod download`
- **Use the latest SDK version** to benefit from security fixes
- **Validate configuration** before initializing clients

### Example Secure Configuration

```go
import (
    "crypto/tls"
    "github.com/logflux-io/logflux-go-sdk/pkg/client"
    "github.com/logflux-io/logflux-go-sdk/pkg/config"
)

// Secure TCP client configuration
cfg := config.Config{
    Address: "logflux-agent.example.com:8080",
    SharedSecret: os.Getenv("LOGFLUX_SECRET"), // From environment
    TLS: &tls.Config{
        MinVersion: tls.VersionTLS12,
        ServerName: "logflux-agent.example.com",
    },
    ConnectionTimeout: 10 * time.Second,
    MaxRetries: 3,
}

client := client.NewTCPClient(cfg)
```

## Known Security Considerations

### BETA Status
- This SDK is in BETA status - use appropriate caution in production environments
- Security features may evolve as the API stabilizes
- Monitor release notes for security-related updates

### Dependencies
- The SDK uses minimal external dependencies to reduce attack surface
- Dependencies are regularly audited for known vulnerabilities
- See `go.mod` for current dependency list

### Agent Communication
- Unix socket communication is preferred for local deployments (more secure)
- TCP communication requires proper network security controls
- Authentication is required for TCP connections but optional for Unix sockets

## Security Updates

Security updates will be communicated through:

- **GitHub Security Advisories**
- **Release Notes** (for non-sensitive updates)
- **Email notifications** (for critical issues, if you've reported vulnerabilities)

## Vulnerability Disclosure Timeline

- **Day 0**: Vulnerability reported
- **Day 1-2**: Initial acknowledgment and triage
- **Day 7**: Initial assessment and response plan
- **Day 30**: Target resolution and patch release
- **Day 37**: Public disclosure (after patch is available)

This timeline may be adjusted based on the severity and complexity of the vulnerability.

## Contact Information

- **Security Email**: security@logflux.io
- **General Issues**: [GitHub Issues](https://github.com/logflux-io/logflux-go-sdk/issues)
- **Documentation**: [docs.logflux.io](https://docs.logflux.io)

## Security Credits

We appreciate the security research community and will acknowledge researchers who responsibly disclose vulnerabilities to help improve LogFlux security.