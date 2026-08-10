import pytest
from unittest.mock import patch, AsyncMock
import mcp.types as types

from src.server import handle_list_tools, handle_call_tool

@pytest.mark.asyncio
async def test_handle_list_tools():
    tools = await handle_list_tools()
    assert len(tools) == 2

    tool_names = [tool.name for tool in tools]
    assert "delegate_task_to_jules" in tool_names
    assert "list_jules_sources" in tool_names

@pytest.mark.asyncio
async def test_handle_call_tool_no_api_key():
    with patch("src.server.jules_client.api_key", None):
        result = await handle_call_tool("list_jules_sources", {})
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "Error: JULES_API_KEY environment variable is not set" in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_list_sources_success():
    with patch("src.server.jules_client.api_key", "test_key"), \
         patch("src.server.jules_client.list_sources", new_callable=AsyncMock) as mock_list_sources:
        mock_list_sources.return_value = {"sources": ["source1", "source2"]}

        result = await handle_call_tool("list_jules_sources", {})
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "Available Jules sources:" in result[0].text
        assert "source1" in result[0].text
        assert "source2" in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_delegate_task_missing_args():
    with patch("src.server.jules_client.api_key", "test_key"):
        result = await handle_call_tool("delegate_task_to_jules", {})
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "Error: Missing source_name or prompt." in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_delegate_task_success():
    with patch("src.server.jules_client.api_key", "test_key"), \
         patch("src.server.jules_client.create_session", new_callable=AsyncMock) as mock_create_session:
        mock_create_session.return_value = {"sessionId": "123"}

        result = await handle_call_tool("delegate_task_to_jules", {"source_name": "src", "prompt": "do something"})
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "Task successfully delegated to Jules!" in result[0].text
        assert "123" in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_unknown_tool():
    with patch("src.server.jules_client.api_key", "test_key"):
        result = await handle_call_tool("unknown_tool", {})
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "Error executing tool unknown_tool: Unknown tool: unknown_tool" in result[0].text

@pytest.mark.asyncio
async def test_handle_call_tool_exception():
    with patch("src.server.jules_client.api_key", "test_key"), \
         patch("src.server.jules_client.list_sources", new_callable=AsyncMock) as mock_list_sources:
        mock_list_sources.side_effect = Exception("API error")

        result = await handle_call_tool("list_jules_sources", {})
        assert len(result) == 1
        assert isinstance(result[0], types.TextContent)
        assert "Error executing tool list_jules_sources: API error" in result[0].text
