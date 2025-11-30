# Security Architecture

This is a read-only data extraction tool with minimal security surface:

| Concern | Approach |
|---------|----------|
| API Access | Public APIs only, no authentication required |
| User Data | None collected or stored |
| Secrets | None required (no API keys) |
| File Permissions | Default user permissions for output files |
| Input Validation | Validate API response shapes, reject malformed data |
| Dependencies | Minimal (1 external dep) to reduce supply chain risk |
