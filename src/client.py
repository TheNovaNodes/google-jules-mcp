"""
Google Jules REST API client (v1alpha).

Resilient HTTP client with:
  - Session reuse (single aiohttp.ClientSession per JulesClient instance)
  - Configurable timeouts
  - Retry with exponential backoff for transient failures (429, 5xx)
"""
from __future__ import annotations

import asyncio
import logging
import os
from typing import Any

import aiohttp
from aiohttp import ClientError, ClientResponseError

try:
    from tenacity import (
        retry,
        retry_if_exception,
        stop_after_attempt,
        wait_exponential_jitter,
    )
    _HAS_TENACITY = True
except ImportError:  # pragma: no cover
    _HAS_TENACITY = False

logger = logging.getLogger(__name__)

# Default timeout: 60s total, 10s connect, 30s sock_read
DEFAULT_TIMEOUT = aiohttp.ClientTimeout(
    total=60, connect=10, sock_connect=10, sock_read=30
)

# Retry configuration
DEFAULT_RETRY_ATTEMPTS = 3
DEFAULT_RETRY_WAIT_MIN = 1  # seconds
DEFAULT_RETRY_WAIT_MAX = 10  # seconds


def _is_retryable_error(exc: Exception) -> bool:
    """Classify whether an exception warrants a retry."""
    if isinstance(exc, ClientResponseError):
        # Retry on rate-limiting and server errors
        return exc.status in (429, 500, 502, 503, 504)
    return isinstance(exc, ClientError)  # Connection errors, timeouts, etc.


class JulesClient:
    """Client for interacting with the Google Jules REST API (v1alpha).

    Manages a single ``aiohttp.ClientSession`` for connection pooling and
    applies timeout + retry policies on every request.
    """

    BASE_URL = "https://jules.googleapis.com/v1alpha"

    def __init__(
        self,
        api_key: str | None = None,
        timeout: aiohttp.ClientTimeout = DEFAULT_TIMEOUT,
        retry_attempts: int = DEFAULT_RETRY_ATTEMPTS,
        retry_wait_min: int = DEFAULT_RETRY_WAIT_MIN,
        retry_wait_max: int = DEFAULT_RETRY_WAIT_MAX,
    ):
        self.api_key = api_key or os.getenv("JULES_API_KEY")
        self._timeout = timeout
        self._retry_attempts = retry_attempts
        self._retry_wait_min = retry_wait_min
        self._retry_wait_max = retry_wait_max

        self._session: aiohttp.ClientSession | None = None
        if not self.api_key:
            logger.warning("JULES_API_KEY is not set.")

    async def _get_session(self) -> aiohttp.ClientSession:
        """Lazily create and reuse a single ``ClientSession``.

        Per aiohttp docs: "Do not create a session per request. Most likely
        you need a single session for the whole lifetime of your application."
        """
        if self._session is None or self._session.closed:
            connector = aiohttp.TCPConnector(
                limit=10,
                limit_per_host=5,
                enable_cleanup_closed=True,
            )
            self._session = aiohttp.ClientSession(
                timeout=self._timeout,
                connector=connector,
            )
        return self._session

    async def _request(
        self,
        method: str,
        endpoint: str,
        json_data: dict | None = None,
    ) -> dict[str, Any]:
        """Make an HTTP request to the Jules API with retry logic.

        Retries on:
          - HTTP 429 (rate limited)
          - HTTP 5xx (server errors)
          - aiohttp ClientError (connection issues, timeouts)
        """
        if not self.api_key:
            raise ValueError("Cannot make request: JULES_API_KEY is missing.")

        headers = {
            "Content-Type": "application/json",
            "x-goog-api-key": self.api_key,
        }
        url = f"{self.BASE_URL}/{endpoint.lstrip('/')}"

        session = await self._get_session()

        if _HAS_TENACITY:
            response = await self._request_with_retry(
                session, method, url, headers, json_data
            )
        else:
            # Fallback: single attempt without retry if tenacity is not installed
            logger.debug("tenacity not available; making single attempt without retry")
            async with session.request(
                method, url, headers=headers, json=json_data
            ) as resp:
                response = await self._handle_response(resp)

        return response

    async def _ensure_session_open(self, session: aiohttp.ClientSession) -> None:
        """Verify the session is usable."""
        if session.closed:
            raise RuntimeError(
                "ClientSession is closed. Call close() and create a new client."
            )

    # -- Retry wrapper --------------------------------------------------

    if _HAS_TENACITY:
        @retry(
            stop=stop_after_attempt(DEFAULT_RETRY_ATTEMPTS),
            wait=wait_exponential_jitter(
                initial=DEFAULT_RETRY_WAIT_MIN,
                max=DEFAULT_RETRY_WAIT_MAX,
            ),
            retry=retry_if_exception(_is_retryable_error),
            reraise=True,
        )
        async def _request_with_retry(
            self,
            session: aiohttp.ClientSession,
            method: str,
            url: str,
            headers: dict,
            json_data: dict | None,
        ) -> dict[str, Any]:
            """Retry-wrapped request handler."""
            async with session.request(
                method, url, headers=headers, json=json_data
            ) as response:
                return await self._handle_response(response)

    # -- Response handler -----------------------------------------------

    @staticmethod
    async def _handle_response(
        response: aiohttp.ClientResponse,
    ) -> dict[str, Any]:
        """Process an HTTP response, raising on error."""
        if not response.ok:
            error_text = await response.text()
            logger.error(
                f"Jules API Error {response.status}: {error_text}"
            )
            response.raise_for_status()
        return await response.json()

    # -- Public API -----------------------------------------------------

    async def list_sources(self) -> dict[str, Any]:
        """List available sources (e.g. connected GitHub repositories)."""
        return await self._request("GET", "/sources")

    async def create_session(
        self,
        source_name: str,
        prompt: str,
        starting_branch: str = "main",
    ) -> dict[str, Any]:
        """Create a new Jules session for a specific source.

        Args:
            source_name: The Jules source identifier (e.g. 'sources/github/owner/repo').
            prompt: Detailed instructions for Jules.
            starting_branch: Git branch to start from (default: 'main').
        """
        payload = {
            "prompt": prompt,
            "sourceContext": {
                "source": source_name,
                "githubRepoContext": {
                    "startingBranch": starting_branch,
                },
            },
        }
        return await self._request("POST", "/sessions", json_data=payload)

    async def close(self) -> None:
        """Clean up the underlying ``ClientSession``.

        Should be called during application shutdown to release
        connection pool resources.
        """
        if self._session and not self._session.closed:
            await self._session.close()
            self._session = None
            logger.debug("JulesClient session closed.")

    def __del__(self):
        """Best-effort cleanup warning if session wasn't closed."""
        if self._session and not self._session.closed:
            logger.warning(
                "JulesClient.__del__ called with an open session. "
                "Use 'await client.close()' for proper cleanup."
            )


# Simple test block to run directly
if __name__ == "__main__":
    from dotenv import load_dotenv

    async def test():
        load_dotenv()
        client = JulesClient()
        if not client.api_key:
            print("Error: JULES_API_KEY is not set in .env")
            return

        print("Testing Jules API...")
        try:
            sources = await client.list_sources()
            print("Successfully fetched sources!")
            print(sources)
        except Exception as e:  # noqa: BLE001
            print(f"Failed to fetch sources: {e}")
        finally:
            await client.close()

    asyncio.run(test())
