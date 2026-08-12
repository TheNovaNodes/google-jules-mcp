"""
Model Context Protocol (MCP) Server for Google Jules AI Coding Agent.
Provides tools for multi-account GitHub source discovery, autonomous session dispatching,
interactive plan approval, follow-up steerage, unified diff retrieval, and activity monitoring.
"""

import os
import json
import asyncio
import logging
from typing import Any, Dict, List, Optional
from mcp.server import Server, NotificationOptions
from mcp.server.models import InitializationOptions
import mcp.types as types
from mcp.server.stdio import stdio_server

from src.router import JulesMultiAccountManager
from src.client import JulesClient
from src.booster import boost_prompt

# Logging configuration
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("google-jules-mcp")

# Initialize Multi-Account Manager
router = JulesMultiAccountManager()
server = Server("google-jules-mcp")

def get_client_for_request(source_name: Optional[str] = None, account_id: Optional[str] = None) -> JulesClient:
    acc = router.resolve_account(source_name=source_name, explicit_account_id=account_id)
    return JulesClient(api_key=acc.api_key)

@server.list_tools()
async def handle_list_tools() -> list[types.Tool]:
    """List all available MCP tools for interacting with Google Jules."""
    return [
        types.Tool(
            name="jules_list_accounts",
            description="List all configured Google Jules / GitHub accounts and routing aliases in the system.",
            inputSchema={
                "type": "object",
                "properties": {},
                "required": []
            }
        ),
        types.Tool(
            name="jules_list_sources",
            description="List all GitHub repositories connected to Google Jules for a specific account or across defaults.",
            inputSchema={
                "type": "object",
                "properties": {
                    "account_id": {
                        "type": "string",
                        "description": "Optional account alias: 'thedoctormes' (DoctorM & Ai) or 'thenovanodes' (TheNovaNodes)."
                    },
                    "page_size": {
                        "type": "integer",
                        "description": "Max number of sources to return (default 100).",
                        "default": 100
                    }
                },
                "required": []
            }
        ),
        types.Tool(
            name="jules_delegate_task",
            description=(
                "Delegate an autonomous coding, refactoring, bug-fixing, or documentation task to Google Jules. "
                "Jules will spin up a Google Cloud VM sandbox, clone the repo, plan changes, run test commands, "
                "apply modifications, and open a Pull Request automatically. Includes prompt boosting & verification contracts."
            ),
            inputSchema={
                "type": "object",
                "properties": {
                    "source_name": {
                        "type": "string",
                        "description": "Repository identifier (e.g. 'sources/github/TheNovaNodes/zakupki-parser-export' or 'thedoctormes-hue/gxlab')."
                    },
                    "prompt": {
                        "type": "string",
                        "description": "Task description, problem statement, or engineering requirement."
                    },
                    "starting_branch": {
                        "type": "string",
                        "description": "Base branch to clone and build upon (default: 'main').",
                        "default": "main"
                    },
                    "require_plan_approval": {
                        "type": "boolean",
                        "description": "If true, Jules will pause after generating the plan and await your approval before making changes.",
                        "default": False
                    },
                    "auto_create_pr": {
                        "type": "boolean",
                        "description": "Whether Jules should automatically push commits and open a GitHub PR upon completion.",
                        "default": True
                    },
                    "target_files": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "Optional list of primary file paths that Jules should focus on."
                    },
                    "test_command": {
                        "type": "string",
                        "description": "Optional explicit test command for Jules to run (e.g. 'pytest tests/ -v' or 'npm test')."
                    },
                    "account_id": {
                        "type": "string",
                        "description": "Optional account override ('thedoctormes' or 'thenovanodes'). Auto-detected from source_name if omitted."
                    }
                },
                "required": ["source_name", "prompt"]
            }
        ),
        types.Tool(
            name="jules_get_session",
            description="Retrieve status, generated outputs, Pull Request links, and metadata for a specific Jules session.",
            inputSchema={
                "type": "object",
                "properties": {
                    "session_id": {
                        "type": "string",
                        "description": "The unique Jules session ID (e.g. '10174540046405012210' or 'sessions/10174540046405012210')."
                    },
                    "account_id": {
                        "type": "string",
                        "description": "Optional account ID if known ('thedoctormes' or 'thenovanodes')."
                    }
                },
                "required": ["session_id"]
            }
        ),
        types.Tool(
            name="jules_list_sessions",
            description="List recent active and completed Jules sessions for an account.",
            inputSchema={
                "type": "object",
                "properties": {
                    "account_id": {
                        "type": "string",
                        "description": "Account ID: 'thedoctormes' or 'thenovanodes'."
                    },
                    "page_size": {
                        "type": "integer",
                        "description": "Number of sessions to retrieve (default 30).",
                        "default": 30
                    }
                },
                "required": []
            }
        ),
        types.Tool(
            name="jules_list_activities",
            description="Get the chronological event timeline of a session (agent thoughts, execution plans, bash outputs, diffs, PRs).",
            inputSchema={
                "type": "object",
                "properties": {
                    "session_id": {
                        "type": "string",
                        "description": "The Jules session ID."
                    },
                    "account_id": {
                        "type": "string",
                        "description": "Optional account ID override."
                    }
                },
                "required": ["session_id"]
            }
        ),
        types.Tool(
            name="jules_approve_plan",
            description="Approve the generated execution plan for a session paused in AWAITING_PLAN_APPROVAL state.",
            inputSchema={
                "type": "object",
                "properties": {
                    "session_id": {
                        "type": "string",
                        "description": "The Jules session ID to approve."
                    },
                    "account_id": {
                        "type": "string",
                        "description": "Optional account ID override."
                    }
                },
                "required": ["session_id"]
            }
        ),
        types.Tool(
            name="jules_send_message",
            description="Send follow-up guidance, feedback, or adjustments to an ongoing Jules session.",
            inputSchema={
                "type": "object",
                "properties": {
                    "session_id": {
                        "type": "string",
                        "description": "The Jules session ID."
                    },
                    "message": {
                        "type": "string",
                        "description": "Follow-up message, prompt adjustment, or feedback."
                    },
                    "account_id": {
                        "type": "string",
                        "description": "Optional account ID override."
                    }
                },
                "required": ["session_id", "message"]
            }
        ),
        types.Tool(
            name="jules_get_diff",
            description="Extract the unified git patch diff from a completed or active Jules session.",
            inputSchema={
                "type": "object",
                "properties": {
                    "session_id": {
                        "type": "string",
                        "description": "The Jules session ID."
                    },
                    "account_id": {
                        "type": "string",
                        "description": "Optional account ID override."
                    }
                },
                "required": ["session_id"]
            }
        )
    ]

@server.call_tool()
async def handle_call_tool(
    name: str, arguments: dict[str, Any] | None
) -> list[types.TextContent | types.ImageContent | types.EmbeddedResource]:
    """Execute Jules MCP tool requests."""
    args = arguments or {}

    try:
        if name == "jules_list_accounts":
            accounts_data = router.list_accounts()
            return [types.TextContent(
                type="text",
                text=f"🏢 Configured Jules Accounts:\n\n{json.dumps(accounts_data, indent=2, ensure_ascii=False)}"
            )]

        elif name == "jules_list_sources":
            acc_id = args.get("account_id")
            page_size = args.get("page_size", 100)

            # If no account specified, list for all accounts
            if not acc_id:
                combined_results = {}
                for acc_key in router.accounts.keys():
                    client = get_client_for_request(account_id=acc_key)
                    res = await client.list_sources(page_size=page_size)
                    combined_results[acc_key] = res.get("sources", [])
                return [types.TextContent(
                    type="text",
                    text=f"📚 Connected GitHub Repositories (All Accounts):\n\n{json.dumps(combined_results, indent=2, ensure_ascii=False)}"
                )]
            else:
                client = get_client_for_request(account_id=acc_id)
                res = await client.list_sources(page_size=page_size)
                return [types.TextContent(
                    type="text",
                    text=f"📚 Connected GitHub Repositories ({acc_id}):\n\n{json.dumps(res, indent=2, ensure_ascii=False)}"
                )]

        elif name == "jules_delegate_task":
            source_name = args.get("source_name")
            raw_prompt = args.get("prompt")
            starting_branch = args.get("starting_branch", "main")
            require_plan_approval = args.get("require_plan_approval", False)
            auto_create_pr = args.get("auto_create_pr", True)
            target_files = args.get("target_files")
            test_command = args.get("test_command")
            account_id = args.get("account_id")

            if not source_name or not raw_prompt:
                return [types.TextContent(type="text", text="❌ Error: source_name and prompt are required.")]

            # Boost the prompt to maximize Jules's reasoning & code execution
            boosted = boost_prompt(
                user_prompt=raw_prompt,
                target_files=target_files,
                test_command=test_command
            )

            client = get_client_for_request(source_name=source_name, account_id=account_id)
            session = await client.create_session(
                source_name=source_name,
                prompt=boosted,
                starting_branch=starting_branch,
                require_plan_approval=require_plan_approval,
                auto_create_pr=auto_create_pr
            )

            session_id = session.get("id") or session.get("name", "").replace("sessions/", "")
            session_url = session.get("url", f"https://jules.google.com/session/{session_id}")

            return [types.TextContent(
                type="text",
                text=(
                    f"🚀 Task successfully delegated to Google Jules!\n\n"
                    f"• Session ID: `{session_id}`\n"
                    f"• Web UI: {session_url}\n"
                    f"• State: `{session.get('state', 'QUEUED')}`\n"
                    f"• Repository: `{source_name}` (branch: `{starting_branch}`)\n"
                    f"• Plan Approval Required: `{require_plan_approval}`\n\n"
                    f"Full Response:\n{json.dumps(session, indent=2, ensure_ascii=False)}"
                )
            )]

        elif name == "jules_get_session":
            session_id = args.get("session_id")
            account_id = args.get("account_id")
            client = get_client_for_request(account_id=account_id)
            session = await client.get_session(session_id)

            # Summarize PR outputs if present
            pr_info = ""
            for out in session.get("outputs", []):
                if "pullRequest" in out:
                    pr = out["pullRequest"]
                    pr_info += f"\n🔗 Pull Request Created: {pr.get('url')}\nTitle: {pr.get('title')}\n"

            return [types.TextContent(
                type="text",
                text=f"📊 Jules Session Status: `{session.get('state')}`{pr_info}\n\n{json.dumps(session, indent=2, ensure_ascii=False)}"
            )]

        elif name == "jules_list_sessions":
            account_id = args.get("account_id")
            page_size = args.get("page_size", 30)
            client = get_client_for_request(account_id=account_id)
            sessions = await client.list_sessions(page_size=page_size)
            return [types.TextContent(
                type="text",
                text=f"📋 Recent Jules Sessions:\n\n{json.dumps(sessions, indent=2, ensure_ascii=False)}"
            )]

        elif name == "jules_list_activities":
            session_id = args.get("session_id")
            account_id = args.get("account_id")
            client = get_client_for_request(account_id=account_id)
            activities = await client.list_activities(session_id)
            return [types.TextContent(
                type="text",
                text=f"📜 Activity Timeline for Session {session_id}:\n\n{json.dumps(activities, indent=2, ensure_ascii=False)}"
            )]

        elif name == "jules_approve_plan":
            session_id = args.get("session_id")
            account_id = args.get("account_id")
            client = get_client_for_request(account_id=account_id)
            res = await client.approve_plan(session_id)
            return [types.TextContent(
                type="text",
                text=f"✅ Execution plan approved for session `{session_id}`. Jules is now executing the implementation."
            )]

        elif name == "jules_send_message":
            session_id = args.get("session_id")
            message = args.get("message")
            account_id = args.get("account_id")
            client = get_client_for_request(account_id=account_id)
            res = await client.send_message(session_id, message)
            return [types.TextContent(
                type="text",
                text=f"💬 Message/feedback sent to Jules session `{session_id}`:\n\n\"{message}\""
            )]

        elif name == "jules_get_diff":
            session_id = args.get("session_id")
            account_id = args.get("account_id")
            client = get_client_for_request(account_id=account_id)
            session = await client.get_session(session_id)

            diffs = []
            for out in session.get("outputs", []):
                patch = out.get("changeSet", {}).get("gitPatch", {}).get("unidiffPatch")
                if patch:
                    diffs.append(patch)

            if not diffs:
                # Also check activities
                acts = await client.list_activities(session_id)
                for a in acts.get("activities", []):
                    for art in a.get("artifacts", []):
                        if "gitPatch" in art:
                            diffs.append(art["gitPatch"].get("unidiffPatch", ""))

            diff_text = "\n\n".join(diffs) if diffs else "No diff found for this session yet."
            return [types.TextContent(
                type="text",
                text=f"📄 Unified Diff for Session {session_id}:\n\n```diff\n{diff_text}\n```"
            )]

        else:
            raise ValueError(f"Unknown MCP tool: {name}")

    except Exception as e:
        logger.exception(f"Error handling MCP tool {name}")
        return [types.TextContent(
            type="text",
            text=f"❌ Error executing tool `{name}`: {str(e)}"
        )]

async def main():
    logger.info("🚀 Starting Google Jules SOTA MCP Server via stdio...")
    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            InitializationOptions(
                server_name="google-jules-mcp",
                server_version="2.0.0",
                capabilities=server.get_capabilities(
                    notification_options=NotificationOptions(),
                    experimental_capabilities={},
                ),
            ),
        )

if __name__ == "__main__":
    asyncio.run(main())
