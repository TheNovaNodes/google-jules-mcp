"""
Tests for the FastMCP-based Jules MCP server.

Covers:
  - Tool listing (2 tools)
  - API key guard
  - list_jules_sources happy path
  - delegate_task_to_jules happy path + arg validation
  - Error handling for unknown tools
  - Lazy client initialization (no session created at import)
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from src.server import (check_jules_status, delegate_task_to_jules,
                        get_jules_client, list_jules_sources, mcp)


@pytest.mark.asyncio
async def test_mcp_server_name():
    """Verify the MCP server is configured with the correct name."""
    assert mcp.name == "google-jules-mcp"


@pytest.mark.asyncio
async def test_tools_are_registered():
    """Verify both tools are registered on the FastMCP instance."""
    tools = await mcp.list_tools()
    tool_names = {t.name for t in tools}
    assert "list_jules_sources" in tool_names
    assert "delegate_task_to_jules" in tool_names
    assert "check_jules_status" in tool_names


@pytest.mark.asyncio
async def test_list_jules_sources_no_api_key():
    """Test that missing API key returns a proper error message."""
    mock_client = MagicMock()
    mock_client.api_key = None

    with patch("src.server.get_jules_client", return_value=mock_client):
        result = await list_jules_sources()

    assert "Error: JULES_API_KEY environment variable is not set" in result


@pytest.mark.asyncio
async def test_list_jules_sources_success():
    """Test successful source listing."""
    mock_client = MagicMock()
    mock_client.api_key = "test_key"
    mock_client.list_sources = AsyncMock(return_value={"sources": ["repo1", "repo2"]})

    with patch("src.server.get_jules_client", return_value=mock_client):
        result = await list_jules_sources()

    mock_client.list_sources.assert_awaited_once()
    assert "Available Jules sources:" in result
    assert "repo1" in result
    assert "repo2" in result


@pytest.mark.asyncio
async def test_list_jules_sources_error():
    """Test error handling in list_jules_sources."""
    mock_client = MagicMock()
    mock_client.api_key = "test_key"
    mock_client.list_sources = AsyncMock(side_effect=Exception("API error"))

    with patch("src.server.get_jules_client", return_value=mock_client):
        result = await list_jules_sources()

    assert "Error executing list_jules_sources: API error" in result


@pytest.mark.asyncio
async def test_delegate_task_missing_args():
    """Test that missing source_name or prompt returns error."""
    mock_client = MagicMock()
    mock_client.api_key = "test_key"

    with patch("src.server.get_jules_client", return_value=mock_client):
        result = await delegate_task_to_jules("", "prompt")
        assert "Error: Missing source_name or prompt." in result

        result = await delegate_task_to_jules("source", "")
        assert "Error: Missing source_name or prompt." in result


@pytest.mark.asyncio
async def test_delegate_task_no_api_key():
    """Test that missing API key returns a proper error message."""
    mock_client = MagicMock()
    mock_client.api_key = None

    with patch("src.server.get_jules_client", return_value=mock_client):
        result = await delegate_task_to_jules("source_name", "prompt")

    assert "Error: JULES_API_KEY environment variable is not set" in result


@pytest.mark.asyncio
async def test_delegate_task_success():
    """Test successful task delegation."""
    mock_client = MagicMock()
    mock_client.api_key = "test_key"
    mock_client.create_session = AsyncMock(return_value={"sessionId": "123"})

    with patch("src.server.get_jules_client", return_value=mock_client):
        result = await delegate_task_to_jules("source_name", "do something")

    mock_client.create_session.assert_awaited_once_with("source_name", "do something")
    assert "Task successfully delegated to Jules!" in result
    assert "123" in result


@pytest.mark.asyncio
async def test_delegate_task_error():
    """Test error handling in delegate_task_to_jules."""
    mock_client = MagicMock()
    mock_client.api_key = "test_key"
    mock_client.create_session = AsyncMock(side_effect=Exception("Session API error"))

    with (
        patch("src/server.get_jules_client", return_value=mock_client)
        if False
        else patch("src.server.get_jules_client", return_value=mock_client)
    ):
        result = await delegate_task_to_jules("source_name", "do something")

    assert "Error executing delegate_task_to_jules: Session API error" in result


@pytest.mark.asyncio
async def test_lazy_client_initialization():
    """Test that get_jules_client lazily creates the client (not at import time)."""
    from src.server import _jules_client

    # After import, the client should be None (lazy)
    assert (
        _jules_client is None
    ), "Client should be None at import time (lazy init), not created eagerly"

    # But get_jules_client should create it
    client = get_jules_client()
    assert client is not None, "get_jules_client should create the client"


@pytest.mark.asyncio
async def test_check_jules_status_success():
    """Test successful status check."""
    mock_client = MagicMock()
    mock_client.api_key = "test_key"
    mock_client.get_session = AsyncMock(return_value={"state": "COMPLETED"})

    with patch("src.server.get_jules_client", return_value=mock_client):
        result = await check_jules_status("session_123")

    mock_client.get_session.assert_awaited_once_with("session_123")
    assert "COMPLETED" in result


@pytest.mark.asyncio
async def test_check_jules_status_missing_args():
    """Test that missing session_id returns error."""
    result = await check_jules_status("")
    assert "Error: Missing session_id" in result
