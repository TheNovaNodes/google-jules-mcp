import pytest
import os
from unittest.mock import patch, MagicMock, AsyncMock
import aiohttp

from src.client import JulesClient

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
    client = JulesClient(api_key="test_key")
    mock_response_data = {"sources": ["source1", "source2"]}

    mock_response = AsyncMock()
    mock_response.ok = True
    mock_response.json.return_value = mock_response_data

    mock_session = MagicMock()
    mock_session.request.return_value.__aenter__.return_value = mock_response

    mock_session_manager = AsyncMock()
    mock_session_manager.__aenter__.return_value = mock_session

    with patch("aiohttp.ClientSession", return_value=mock_session_manager):
        result = await client.list_sources()
        assert result == mock_response_data

@pytest.mark.asyncio
async def test_jules_client_create_session():
    client = JulesClient(api_key="test_key")
    mock_response_data = {"sessionId": "123"}

    mock_response = AsyncMock()
    mock_response.ok = True
    mock_response.json.return_value = mock_response_data

    mock_session = MagicMock()
    mock_session.request.return_value.__aenter__.return_value = mock_response

    mock_session_manager = AsyncMock()
    mock_session_manager.__aenter__.return_value = mock_session

    with patch("aiohttp.ClientSession", return_value=mock_session_manager):
        result = await client.create_session("source_test", "test prompt")
        assert result == mock_response_data

        # Verify call arguments
        mock_session.request.assert_called_once()
        args, kwargs = mock_session.request.call_args
        assert args[0] == "POST"
        assert args[1] == "https://jules.googleapis.com/v1alpha/sessions"
        assert kwargs["headers"] == {
            "Content-Type": "application/json",
            "x-goog-api-key": "test_key"
        }
        assert kwargs["json"]["prompt"] == "test prompt"
        assert kwargs["json"]["sourceContext"]["source"] == "source_test"

@pytest.mark.asyncio
async def test_jules_client_request_failure():
    client = JulesClient(api_key="test_key")

    mock_response = AsyncMock()
    mock_response.ok = False
    mock_response.status = 400
    mock_response.text.return_value = "Bad Request"

    def side_effect():
        raise aiohttp.ClientResponseError(
            request_info=MagicMock(),
            history=(),
            status=400,
            message="Bad Request"
        )
    mock_response.raise_for_status = side_effect

    mock_session = MagicMock()
    mock_session.request.return_value.__aenter__.return_value = mock_response

    mock_session_manager = AsyncMock()
    mock_session_manager.__aenter__.return_value = mock_session

    with patch("aiohttp.ClientSession", return_value=mock_session_manager):
        with pytest.raises(aiohttp.ClientResponseError):
            await client.list_sources()
