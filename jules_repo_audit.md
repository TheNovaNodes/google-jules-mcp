# Google Jules MCP Server Repository Audit

## 1. Tech Stack
*   **Language:** Python (>=3.10)
*   **Manifest File:** `pyproject.toml`
*   **Key Frameworks & Dependencies:**
    *   `mcp` (>=1.2.0, <2.0.0) - Model Context Protocol SDK
    *   `aiohttp` (>=3.0.0) - Asynchronous HTTP client/server framework
    *   `pydantic` (>=2.0.0) - Data validation and settings management
    *   `python-dotenv` (>=1.0.0) - Read key-value pairs from a `.env` file
    *   `tenacity` (>=8.0.0) - Retrying library
*   **Development Dependencies:**
    *   `pytest` (>=7.0) - Testing framework
    *   `pytest-asyncio` (>=0.21) - Pytest plugin for testing asyncio code

## 2. Verification & Build Commands
*   **Environment Setup:**
    ```bash
    python3 -m venv .venv
    source .venv/bin/activate
    pip install -e .
    cp .env.example .env # Ensure to add your JULES_API_KEY
    ```
*   **Running Tests:**
    ```bash
    pytest tests/
    ```
    *(Note: Pytest with the pytest-asyncio plugin is used to handle asynchronous tests.)*
*   **Running the Project:**
    ```bash
    python src/main.py
    ```
    Alternatively, using the installed script:
    ```bash
    google-jules-mcp
    ```
*   **Running Linters and Formatters:**
    The contributing guidelines specify that code must be formatted and pass linting checks, but no explicit shell commands (like `black`, `flake8`, or `ruff`) are defined in the `pyproject.toml` or `README.md`.

## 3. Project Structure
```
.
├── AGENTS.md         # AI Agent specific instructions
├── CONTRIBUTING.md   # Guidelines for contributing to the project
├── README.md         # Main project documentation and overview
├── docs/             # Additional documentation and specifications
│   └── specs/
├── pyproject.toml    # Python project configuration and dependency file
├── src/              # Application source code
│   ├── __init__.py
│   ├── client.py     # Asynchronous API client implementation using aiohttp
│   └── server.py     # MCP Server implementation using fastmcp
└── tests/            # Test suite
    ├── __init__.py
    ├── test_client.py
    └── test_server.py
```

## 4. Developer Conventions
*   **Code Quality & Style (`CONTRIBUTING.md`):**
    *   Use type hints wherever possible to improve readability and maintainability.
    *   All new features or bug fixes must be covered by appropriate unit tests.
    *   Code must be formatted correctly and pass linting checks.
*   **Pull Requests (`CONTRIBUTING.md`):**
    *   Do not include issue numbers in the PR title.
    *   Verify that all status checks are passing after submission.
*   **Architectural & Security Rules (`AGENTS.md`):**
    *   **API Keys:** Never hardcode the `JULES_API_KEY` in the source code. It must always be read from environment variables.
    *   **Async Programming:** `src/client.py` uses `aiohttp`. Developers must manage `ClientSession` correctly and handle standard HTTP errors.
    *   **MCP Standard:** The server uses `mcp.server.fastmcp`. Any new tools should be decorated with `@mcp.tool()`.
    *   **Task Delegation:** Always call `list_jules_sources` before `delegate_task_to_jules` to get the correct repository format. Prompts must contain clear and comprehensive instructions.
