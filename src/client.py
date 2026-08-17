"""
High-Performance Async Client for Google Jules REST API (v1alpha).
Supports sources, sessions, multi-turn messaging, plan approvals, activity streaming, diff extraction, and cancellation.
"""

import asyncio
import logging
import aiohttp
from typing import Dict, Any, Optional, List

logger = logging.getLogger("jules-client")

class JulesClient:
    """Asynchronous client for communicating with Google Jules REST API."""

    BASE_URL = "https://jules.googleapis.com/v1alpha"

    def __init__(self, api_key: str):
        if not api_key:
            raise ValueError("JulesClient requires a valid non-empty API key.")
        self.api_key = api_key

    async def _request(
        self,
        method: str,
        endpoint: str,
        params: Optional[Dict[str, Any]] = None,
        json_data: Optional[Dict[str, Any]] = None,
        max_retries: int = 4
    ) -> Dict[str, Any]:
        headers = {
            "Content-Type": "application/json",
            "x-goog-api-key": self.api_key
        }
        url = f"{self.BASE_URL}/{endpoint.lstrip('/')}"

        timeout = aiohttp.ClientTimeout(total=60)
        delay = 1.0

        for attempt in range(1, max_retries + 1):
            try:
                async with aiohttp.ClientSession(timeout=timeout) as session:
                    async with session.request(method, url, headers=headers, params=params, json=json_data) as response:
                        if response.status in (429, 500, 502, 503, 504) and attempt < max_retries:
                            logger.warning(
                                f"Jules API [{method} {endpoint}] hit HTTP {response.status} (attempt {attempt}/{max_retries}). "
                                f"Retrying in {delay:.1f}s..."
                            )
                            await asyncio.sleep(delay)
                            delay *= 2.0
                            continue

                        if not response.ok:
                            error_text = await response.text()
                            logger.error(f"Jules API [{method} {endpoint}] failed with HTTP {response.status}: {error_text}")
                            try:
                                err_json = await response.json()
                                msg = err_json.get("error", {}).get("message", error_text)
                            except Exception:
                                msg = error_text
                            raise RuntimeError(f"Google Jules API Error ({response.status}): {msg}")
                        return await response.json()
            except (aiohttp.ClientError, asyncio.TimeoutError) as e:
                if attempt < max_retries:
                    logger.warning(f"Jules API [{method} {endpoint}] network error: {e}. Retrying in {delay:.1f}s...")
                    await asyncio.sleep(delay)
                    delay *= 2.0
                else:
                    raise RuntimeError(f"Google Jules API Connection Failed after {max_retries} attempts: {e}")

    # --- Sources ---
    async def list_sources(self, page_size: int = 100, page_token: Optional[str] = None) -> Dict[str, Any]:
        params = {"pageSize": page_size}
        if page_token:
            params["pageToken"] = page_token
        return await self._request("GET", "/sources", params=params)

    async def get_source(self, source_name: str) -> Dict[str, Any]:
        clean_name = source_name.lstrip("/")
        return await self._request("GET", f"/{clean_name}")

    # --- Sessions ---
    async def list_sessions(self, page_size: int = 50, page_token: Optional[str] = None) -> Dict[str, Any]:
        params = {"pageSize": page_size}
        if page_token:
            params["pageToken"] = page_token
        return await self._request("GET", "/sessions", params=params)

    async def create_session(
        self,
        source_name: str,
        prompt: str,
        title: Optional[str] = None,
        starting_branch: str = "main",
        require_plan_approval: bool = False,
        auto_create_pr: bool = True
    ) -> Dict[str, Any]:
        # Ensure source begins with sources/
        if not source_name.startswith("sources/"):
            source_name = f"sources/github/{source_name.lstrip('/')}"

        payload: Dict[str, Any] = {
            "prompt": prompt,
            "sourceContext": {
                "source": source_name,
                "githubRepoContext": {
                    "startingBranch": starting_branch
                }
            }
        }
        if title:
            payload["title"] = title
        if require_plan_approval:
            payload["requirePlanApproval"] = True
        if auto_create_pr:
            payload["automationMode"] = "AUTO_CREATE_PR"

        return await self._request("POST", "/sessions", json_data=payload)

    async def get_session(self, session_id: str) -> Dict[str, Any]:
        clean_id = session_id.lstrip("/")
        if not clean_id.startswith("sessions/"):
            clean_id = f"sessions/{clean_id}"
        return await self._request("GET", f"/{clean_id}")

    async def delete_session(self, session_id: str) -> Dict[str, Any]:
        clean_id = session_id.lstrip("/")
        if not clean_id.startswith("sessions/"):
            clean_id = f"sessions/{clean_id}"
        return await self._request("DELETE", f"/{clean_id}")

    async def approve_plan(self, session_id: str) -> Dict[str, Any]:
        clean_id = session_id.lstrip("/")
        if not clean_id.startswith("sessions/"):
            clean_id = f"sessions/{clean_id}"
        return await self._request("POST", f"/{clean_id}:approvePlan", json_data={})

    async def send_message(self, session_id: str, prompt: str) -> Dict[str, Any]:
        clean_id = session_id.lstrip("/")
        if not clean_id.startswith("sessions/"):
            clean_id = f"sessions/{clean_id}"
        return await self._request("POST", f"/{clean_id}:sendMessage", json_data={"prompt": prompt})

    # --- Activities & Diffs ---
    async def list_activities(self, session_id: str, page_size: int = 100, page_token: Optional[str] = None) -> Dict[str, Any]:
        clean_id = session_id.lstrip("/")
        if not clean_id.startswith("sessions/"):
            clean_id = f"sessions/{clean_id}"
        params = {"pageSize": page_size}
        if page_token:
            params["pageToken"] = page_token
        return await self._request("GET", f"/{clean_id}/activities", params=params)
