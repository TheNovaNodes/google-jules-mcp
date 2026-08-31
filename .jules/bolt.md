## 2026-08-31 - [Tenacity Retry Optimization]
**Learning:** `retry_if_exception_type((ClientResponseError, ClientError))` in tenacity will retry every exception that inherits from `ClientError`. Because aiohttp's `ClientResponseError` inherits from `ClientError`, this meant all HTTP errors (even non-retryable 400 or 404s) were being retried, wasting significant time and requests.
**Action:** Use a custom exception checker using `retry_if_exception(custom_func)` for aiohttp clients instead of exception types if you need to filter by status codes.
