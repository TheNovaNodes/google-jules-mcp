import os
import asyncio
import logging
from typing import Any
from mcp.server import Server, NotificationOptions
from mcp.server.models import InitializationOptions
import mcp.types as types
from mcp.server.stdio import stdio_server

from src.client import JulesClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("jules-mcp-server")

server = Server("google-jules-mcp")

# JulesClient will be instantiated when the tool is called to ensure it picks up env vars
# Or we can initialize it globally if env vars are passed to the MCP server process.
jules_client = JulesClient()

@server.list_tools()
async def handle_list_tools() -> list[types.Tool]:
    """List available tools for Jules MCP Server."""
    return [
        types.Tool(
            name="delegate_task_to_jules",
            description=(
                "Delegate a long-running, asynchronous task to Google Jules. "
                "Use this tool when a coding task involves significant refactoring, "
                "large-scale testing, or complex feature implementation across multiple files. "
                "Jules will clone the repository, run the task in a Google Cloud VM, "
                "and propose a Pull Request. This prevents the local chat from blocking."
            ),
            inputSchema={
                "type": "object",
                "properties": {
                    "source_name": {
                        "type": "string",
                        "description": "The name of the GitHub source connected to Jules (e.g. 'sources/github/owner/repo')."
                    },
                    "prompt": {
                        "type": "string",
                        "description": "A detailed explanation of what Jules should do in the repository."
                    }
                },
                "required": ["source_name", "prompt"]
            }
        ),
        types.Tool(
            name="list_jules_sources",
            description="List all GitHub repositories connected to the user's Jules account. Use this to find the correct source_name for delegation.",
            inputSchema={
                "type": "object",
                "properties": {},
                "required": []
            }
        )
    ]

@server.call_tool()
async def handle_call_tool(
    name: str, arguments: dict[str, Any] | None
) -> list[types.TextContent | types.ImageContent | types.EmbeddedResource]:
    """Handle tool execution requests."""
    if not jules_client.api_key:
        return [types.TextContent(
            type="text", 
            text="Error: JULES_API_KEY environment variable is not set for the MCP server."
        )]

    try:
        if name == "list_jules_sources":
            sources = await jules_client.list_sources()
            return [types.TextContent(
                type="text",
                text=f"Available Jules sources: {sources}"
            )]

        elif name == "delegate_task_to_jules":
            source_name = arguments.get("source_name")
            prompt = arguments.get("prompt")
            
            if not source_name or not prompt:
                return [types.TextContent(type="text", text="Error: Missing source_name or prompt.")]
                
            session_result = await jules_client.create_session(source_name, prompt)
            return [types.TextContent(
                type="text",
                text=f"Task successfully delegated to Jules!\n\nDetails: {session_result}"
            )]
            
        else:
            raise ValueError(f"Unknown tool: {name}")

    except Exception as e:
        logger.exception(f"Tool {name} failed.")
        return [types.TextContent(type="text", text=f"Error executing tool {name}: {str(e)}")]


async def main():
    logger.info("Starting Google Jules MCP Server via stdio...")
    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            InitializationOptions(
                server_name="google-jules-mcp",
                server_version="0.1.0",
                capabilities=server.get_capabilities(
                    notification_options=NotificationOptions(),
                    experimental_capabilities={},
                ),
            ),
        )

if __name__ == "__main__":
    asyncio.run(main())
