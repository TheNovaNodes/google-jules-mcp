## 2024-05-18 - [Prevent Information Leakage in Error Logs]
**Vulnerability:** The HTTP client logged the full raw error response text (`await response.text()`) from external API requests upon a non-200 status code.
**Learning:** This exposes potential sensitive data, stack traces, or internal server details from the remote service directly into local logs, creating a security risk (Information Exposure through logging).
**Prevention:** Avoid logging complete `error_text` or raw payloads from external services in error handling routines. Instead, log the standard HTTP status code and reason (`response.status` and `response.reason`) to diagnose the failure without leaking sensitive upstream information.
