import pytest
from unittest.mock import patch, AsyncMock, MagicMock
import mcp.types as types

from src.server import handle_list_tools, handle_call_tool

@pytest.mark.asyncio
async def test_handle_list_tools():
    tools = await handle_list_tools()
    assert len(tools) >= 8

    tool_names = [tool.name for tool in tools]
    assert "jules_delegate_task" in tool_names
    assert "jules_list_sources" in tool_names
    assert "jules_list_accounts" in tool_names

@pytest.mark.asyncio
async def test_handle_call_tool_list_accounts():
    result = await handle_call_tool("jules_list_accounts", {})
    assert len(result) == 1
    assert isinstance(result[0], types.TextContent)
    assert "Configured Jules Accounts" in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_list_sources_success():
    mock_client = AsyncMock()
    mock_client.list_sources.return_value = {"sources": [{"name": "sources/github/TheNovaNodes/test", "id": "test"}]}

    with patch("src.server.get_client_for_request", return_value=mock_client):
        result = await handle_call_tool("jules_list_sources", {})
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "sources/github/TheNovaNodes/test" in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_delegate_task_success():
    mock_client = AsyncMock()
    mock_client.create_session.return_value = {"name": "sessions/123", "id": "123"}

    with patch("src.server.get_client_for_request", return_value=mock_client):
        result = await handle_call_tool("jules_delegate_task", {
            "source_name": "sources/github/TheNovaNodes/test",
            "prompt": "Fix bug"
        })
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "Task successfully delegated to Google Jules!" in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_unknown_tool():
    result = await handle_call_tool("unknown_tool", {})
    assert len(result) == 1
    assert isinstance(result[0], types.TextContent)
    assert "Unknown MCP tool" in result[0].text


