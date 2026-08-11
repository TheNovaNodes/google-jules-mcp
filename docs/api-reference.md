# API & Data Flow Reference

This document outlines the tools exposed via the Model Context Protocol (MCP) and the underlying REST API client methods used to communicate with the Google Jules API.

## MCP Tools (Server-Side)

These tools are exposed to the local AI agent via standard I/O streams using the MCP protocol.

### `list_jules_sources`
Retrieves a list of all GitHub repositories connected and authorized to the user's Jules account.

| Property | Type | Description |
| :--- | :--- | :--- |
| **Inputs** | `object` | No parameters required. |
| **Outputs** | `list[TextContent]` | A formatted string containing the raw JSON response of available sources. |
| **Errors** | `TextContent` | Returns an error message if the `JULES_API_KEY` is missing or if the API request fails. |

### `delegate_task_to_jules`
Delegates a long-running, complex coding or architectural task to Google Jules. Jules will clone the repository, run the task in a Google Cloud VM, and propose a Pull Request.

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `source_name` | `string` | Yes | The name of the GitHub source connected to Jules (e.g., `sources/github/owner/repo`). |
| `prompt` | `string` | Yes | A detailed explanation of what Jules should do in the repository. |

| Property | Type | Description |
| :--- | :--- | :--- |
| **Outputs** | `list[TextContent]` | A success message including the session details returned by the API. |
| **Errors** | `TextContent` | Returns an error if parameters are missing, the API key is unset, or the API request fails. |

---

## Jules REST API Client (`src/client.py`)

The `JulesClient` handles the sanitization, formulation, and transformation of data between the MCP server and the remote Google Jules API (`https://jules.googleapis.com/v1alpha`).

### Data Transformation & Sanitization
- **Authentication**: The client automatically injects the `JULES_API_KEY` (sourced from the environment) into the `x-goog-api-key` header for every request.
- **Payload Construction**: Arguments passed from the MCP tools are transformed into the specific JSON schema required by Jules. For example, `delegate_task_to_jules` parameters are nested into `{"prompt": ..., "sourceContext": {"source": ...}}`.
- **Response Parsing**: The raw JSON response from the aiohttp `ClientSession` is parsed into Python dictionaries using `await response.json()`.
- **Error Handling**: Non-2xx HTTP responses are caught, the error text is extracted and logged, and `response.raise_for_status()` is called to propagate the error back to the MCP server.

### Client Methods

#### `list_sources()`
Makes a `GET` request to the `/sources` endpoint.

| Returns | Description |
| :--- | :--- |
| `Dict[str, Any]` | The parsed JSON response containing available sources. |

#### `create_session(source_name: str, prompt: str)`
Makes a `POST` request to the `/sessions` endpoint.

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `source_name` | `str` | The target repository identifier. |
| `prompt` | `str` | The task instructions. |

| Returns | Description |
| :--- | :--- |
| `Dict[str, Any]` | The parsed JSON response containing the newly created session details. |
