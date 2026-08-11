"""
Tests for JulesClient.

Covers:
  - API key handling (env + explicit)
  - Session reuse across requests (regression for #3)
  - Error handling (no key, HTTP errors)
  - Retry classification logic (unit-level)
"""
import os
from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
import pytest

from src.client import JulesClient, _is_retryable_error


def _make_async_mock_session(response):
    """Build a MagicMock that quacks like an aiohttp session for our retry path.

    ``session.request(...)`` must return something usable in ``async with`` —
    i.e. an async context manager yielding ``response``.
    """
    cm = MagicMock()
    cm.__aenter__ = AsyncMock(return_value=response)
    cm.__aexit__ = AsyncMock(return_value=None)
    mock_session = MagicMock()
    mock_session.closed = False
    mock_session.request.return_value = cm
    return mock_session


def test_jules_client_init_no_key():
    with patch.dict(os.environ, clear=True):
        client = JulesClient()
        assert client.api_key is None


def test_jules_client_init_with_key():
    with patch.dict(os.environ, {"JULES_API_KEY": "test_key"}):
        client = JulesClient()
        assert client.api_key == "test_key"


@pytest.mark.asyncio
async def test_jules_client_request_no_key():
    client = JulesClient(api_key=None)
    with pytest.raises(ValueError, match="Cannot make request: JULES_API_KEY is missing."):
        await client._request("GET", "/test")


@pytest.mark.asyncio
async def test_jules_client_list_sources():
    """Test that list_sources sends correct request with headers."""
    client = JulesClient(api_key="test_key")

    mock_response = MagicMock()
    mock_response.ok = True
    mock_response.json = AsyncMock(return_value={"sources": ["source1", "source2"]})

    mock_session = _make_async_mock_session(mock_response)

    with patch.object(client, "_get_session", new_callable=AsyncMock, return_value=mock_session):
        result = await client.list_sources()
        assert result == {"sources": ["source1", "source2"]}

        mock_session.request.assert_called_once()
        args, kwargs = mock_session.request.call_args
        assert args[0] == "GET"
        assert args[1] == "https://jules.googleapis.com/v1alpha/sources"
        assert kwargs["headers"] == {
            "Content-Type": "application/json",
            "x-goog-api-key": "test_key",
        }


@pytest.mark.asyncio
async def test_jules_client_create_session():
    """Test that create_session sends correct POST with payload."""
    client = JulesClient(api_key="test_key")

    mock_response = MagicMock()
    mock_response.ok = True
    mock_response.json = AsyncMock(return_value={"sessionId": "123"})

    mock_session = _make_async_mock_session(mock_response)

    with patch.object(client, "_get_session", new_callable=AsyncMock, return_value=mock_session):
        result = await client.create_session("source_test", "test prompt")
        assert result == {"sessionId": "123"}

        mock_session.request.assert_called_once()
        args, kwargs = mock_session.request.call_args
        assert args[0] == "POST"
        assert args[1] == "https://jules.googleapis.com/v1alpha/sessions"
        assert kwargs["headers"] == {
            "Content-Type": "application/json",
            "x-goog-api-key": "test_key",
        }
        assert kwargs["json"]["prompt"] == "test prompt"
        assert kwargs["json"]["sourceContext"]["source"] == "source_test"
        assert kwargs["json"]["sourceContext"]["githubRepoContext"]["startingBranch"] == "main"


@pytest.mark.asyncio
async def test_jules_client_create_session_custom_branch():
    """Test that custom starting_branch is passed correctly."""
    client = JulesClient(api_key="test_key")

    mock_response = MagicMock()
    mock_response.ok = True
    mock_response.json = AsyncMock(return_value={"ok": True})

    mock_session = _make_async_mock_session(mock_response)

    with patch.object(client, "_get_session", new_callable=AsyncMock, return_value=mock_session):
        await client.create_session("src", "prompt", starting_branch="dev")

        _, kwargs = mock_session.request.call_args
        assert kwargs["json"]["sourceContext"]["githubRepoContext"]["startingBranch"] == "dev"


@pytest.mark.asyncio
async def test_jules_client_request_failure():
    """Test that HTTP errors are raised."""
    client = JulesClient(api_key="test_key")

    mock_response = MagicMock()
    mock_response.ok = False
    mock_response.status = 400
    mock_response.text = AsyncMock(return_value="Bad Request")

    def raise_error():
        raise aiohttp.ClientResponseError(
            request_info=MagicMock(),
            history=(),
            status=400,
            message="Bad Request",
        )

    mock_response.raise_for_status = raise_error

    mock_session = _make_async_mock_session(mock_response)

    with patch.object(client, "_get_session", new_callable=AsyncMock, return_value=mock_session), \
         pytest.raises(aiohttp.ClientResponseError):
        await client.list_sources()


@pytest.mark.asyncio
async def test_session_is_reused_across_requests():
    """Regression test for #3: _get_session should cache and reuse the session."""
    client = JulesClient(api_key="test_key")

    mock_response = MagicMock()
    mock_response.ok = True
    mock_response.json = AsyncMock(return_value={"sources": []})

    constructed_sessions = []

    def fake_session_constructor(*args, **kwargs):
        sess = _make_async_mock_session(mock_response)
        sess.closed = False
        constructed_sessions.append(sess)
        return sess

    with patch("aiohttp.ClientSession", side_effect=fake_session_constructor):
        await client.list_sources()
        await client.list_sources()

    assert len(constructed_sessions) == 1, (
        f"Expected 1 ClientSession construction, got {len(constructed_sessions)}"
    )


@pytest.mark.asyncio
async def test_session_created_lazily():
    """Test that session is not created at client init time (lazy init)."""
    client = JulesClient(api_key="test_key")
    assert client._session is None, (
        "Session should not be created at init time (lazy init), not created eagerly"
    )


@pytest.mark.asyncio
async def test_close_session():
    """Test that close() properly cleans up the session."""
    client = JulesClient(api_key="test_key")

    mock_session = AsyncMock()
    mock_session.closed = False
    client._session = mock_session

    await client.close()

    mock_session.close.assert_awaited_once()
    assert client._session is None


@pytest.mark.asyncio
async def test_close_when_no_session():
    """Test that close() is safe when no session exists."""
    client = JulesClient(api_key="test_key")
    assert client._session is None
    await client.close()


@pytest.mark.asyncio
async def test_get_session_reuses_existing():
    """Test _get_session returns the same session on subsequent calls."""
    client = JulesClient(api_key="test_key")

    mock_session = AsyncMock()
    mock_session.closed = False

    with patch.object(aiohttp.ClientSession, "__init__", return_value=None):
        await client._get_session()
        client._session = mock_session
        session2 = await client._get_session()

    assert session2 is mock_session


@pytest.mark.asyncio
async def test_get_session_recreates_if_closed():
    """Test _get_session creates a new session if the previous one is closed."""
    client = JulesClient(api_key="test_key")

    old_session = AsyncMock()
    old_session.closed = True
    client._session = old_session

    new_session = AsyncMock()
    new_session.closed = False

    with patch("aiohttp.ClientSession", return_value=new_session):
        result = await client._get_session()

    assert result is new_session
    assert client._session is new_session


def test_is_retryable_error():
    """Test the _is_retryable_error classification function."""
    # 429 - retryable
    err_429 = aiohttp.ClientResponseError(
        request_info=MagicMock(), history=(), status=429, message="Too Many Requests"
    )
    assert _is_retryable_error(err_429) is True

    # 500 - retryable
    err_500 = aiohttp.ClientResponseError(
        request_info=MagicMock(), history=(), status=500, message="Internal Server Error"
    )
    assert _is_retryable_error(err_500) is True

    # 503 - retryable
    err_503 = aiohttp.ClientResponseError(
        request_info=MagicMock(), history=(), status=503, message="Service Unavailable"
    )
    assert _is_retryable_error(err_503) is True

    # 400 - NOT retryable
    err_400 = aiohttp.ClientResponseError(
        request_info=MagicMock(), history=(), status=400, message="Bad Request"
    )
    assert _is_retryable_error(err_400) is False

    # 404 - NOT retryable
    err_404 = aiohttp.ClientResponseError(
        request_info=MagicMock(), history=(), status=404, message="Not Found"
    )
    assert _is_retryable_error(err_404) is False

    # Connection error - retryable (ClientError)
    conn_err = aiohttp.ClientConnectionError("connection refused")
    assert _is_retryable_error(conn_err) is True
