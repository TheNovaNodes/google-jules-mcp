import pytest
import os
from unittest.mock import patch, MagicMock, AsyncMock
import aiohttp

from src.client import JulesClient

def test_jules_client_init_no_key():
    with pytest.raises(ValueError, match="JulesClient requires a valid non-empty API key."):
        JulesClient(api_key="")

def test_jules_client_init_with_key():
    client = JulesClient(api_key="test_key")
    assert client.api_key == "test_key"

@pytest.mark.asyncio
async def test_jules_client_request_no_key():
    with pytest.raises(ValueError, match="JulesClient requires a valid non-empty API key."):
        JulesClient(api_key="")

@pytest.mark.asyncio
async def test_jules_client_list_sources():
    client = JulesClient(api_key="test_key")
    mock_response_data = {"sources": ["source1", "source2"]}

    mock_response = AsyncMock()
    mock_response.status = 200
    mock_response.ok = True
    mock_response.json.return_value = mock_response_data

    mock_session = MagicMock()
    mock_session.request.return_value.__aenter__.return_value = mock_response

    mock_session_manager = MagicMock()
    mock_session_manager.__aenter__.return_value = mock_session
    mock_session_manager.__aexit__.return_value = False

    with patch("aiohttp.ClientSession", return_value=mock_session_manager):
        result = await client.list_sources()
        assert result == mock_response_data

@pytest.mark.asyncio
async def test_jules_client_create_session():
    client = JulesClient(api_key="test_key")
    mock_response_data = {"sessionId": "123"}

    mock_response = AsyncMock()
    mock_response.status = 200
    mock_response.ok = True
    mock_response.json.return_value = mock_response_data

    mock_session = MagicMock()
    mock_session.request.return_value.__aenter__.return_value = mock_response

    mock_session_manager = MagicMock()
    mock_session_manager.__aenter__.return_value = mock_session
    mock_session_manager.__aexit__.return_value = False

    with patch("aiohttp.ClientSession", return_value=mock_session_manager):
        result = await client.create_session("source_test", "test prompt")
        assert result == mock_response_data

@pytest.mark.asyncio
async def test_jules_client_request_failure():
    client = JulesClient(api_key="test_key")

    mock_response = AsyncMock()
    mock_response.status = 400
    mock_response.ok = False
    mock_response.text.return_value = "Bad Request"

    mock_session = MagicMock()
    mock_session.request.return_value.__aenter__.return_value = mock_response

    mock_session_manager = MagicMock()
    mock_session_manager.__aenter__.return_value = mock_session
    mock_session_manager.__aexit__.return_value = False

    with patch("aiohttp.ClientSession", return_value=mock_session_manager):
        with pytest.raises(RuntimeError, match="Google Jules API Error \\(400\\)"):
            await client.list_sources()

