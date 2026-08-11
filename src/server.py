"""
MCP Server for Google Jules AI Agent.

Uses FastMCP (high-level MCP API) with proper lifecycle management:
  - Lazy client initialization (env vars read at first request, not import time)
  - Session cleanup on shutdown via lifespan hooks
"""
import logging
from contextlib import asynccontextmanager

from mcp.server.fastmcp import FastMCP

from src.client import JulesClient

logger = logging.getLogger("jules-mcp-server")

# --- Lifecycle-managed client (lazy init) -----------------------------------

_jules_client: JulesClient | None = None


@asynccontextmanager
async def lifespan(app: FastMCP):
    """Manage the JulesClient lifecycle.

    - Client is None at startup (lazy creation on first request).
    - Env vars read at runtime (not import time) — fixes #5.
    - Session is cleanly closed on shutdown — fixes #3/#5.
    """
    global _jules_client
    _jules_client = None  # Ensure fresh init on startup
    logger.info("Google Jules MCP Server starting (FastMCP)...")
    yield
    # Cleanup on shutdown
    if _jules_client is not None:
        await _jules_client.close()
        logger.info("JulesClient session closed on shutdown.")


mcp = FastMCP("google-jules-mcp", lifespan=lifespan)


def get_jules_client() -> JulesClient:
    """Lazily instantiate JulesClient on first use.

    This ensures JULES_API_KEY is read from the environment at request time,
    not at import time — fixing the singleton/env-read-at-import issue (#5).
    """
    global _jules_client
    if _jules_client is None:
        _jules_client = JulesClient()
    return _jules_client


# --- Tools -------------------------------------------------------------------


@mcp.tool()
async def list_jules_sources() -> str:
    """List all GitHub repositories connected to the user's Jules account.

    Use this to find the correct source_name for delegation.
    """
    client = get_jules_client()
    if not client.api_key:
        return "Error: JULES_API_KEY environment variable is not set for the MCP server."

    try:
        sources = await client.list_sources()
        return f"Available Jules sources: {sources}"
    except Exception as e:
        logger.exception("list_jules_sources failed.")
        return f"Error executing list_jules_sources: {e}"


@mcp.tool()
async def delegate_task_to_jules(source_name: str, prompt: str) -> str:
    """Delegate a long-running, asynchronous task to Google Jules.

    Use this tool when a coding task involves significant refactoring,
    large-scale testing, or complex feature implementation across multiple files.
    Jules will clone the repository, run the task in a Google Cloud VM,
    and propose a Pull Request. This prevents the local chat from blocking.

    Args:
        source_name: The name of the GitHub source connected to Jules
            (e.g. 'sources/github/owner/repo').
        prompt: A detailed explanation of what Jules should do in the repository.
    """
    if not source_name or not prompt:
        return "Error: Missing source_name or prompt."

    client = get_jules_client()
    if not client.api_key:
        return "Error: JULES_API_KEY environment variable is not set for the MCP server."

    try:
        session_result = await client.create_session(source_name, prompt)
        return f"Task successfully delegated to Jules!\n\nDetails: {session_result}"
    except Exception as e:
        logger.exception("delegate_task_to_jules failed.")
        return f"Error executing delegate_task_to_jules: {e}"


def main():
    """Entry point — run the MCP server in stdio mode."""
    mcp.run()


if __name__ == "__main__":
    main()
