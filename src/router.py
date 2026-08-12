"""
Multi-Account & Multi-Tenant Router for Google Jules API.
Supports automatic routing based on repository owner name, explicit account alias,
and local configuration files / environment variables.
"""

import os
import re
import logging
from pathlib import Path
from dataclasses import dataclass, field
from typing import Dict, List, Optional
from dotenv import load_dotenv

logger = logging.getLogger("jules-router")

# Load environment variables from standard locations
load_dotenv()
load_dotenv(Path("/root/lab/.env"))
load_dotenv(Path("/root/lab/thenovanodes/antigravity-telegram-agent/.env"))

@dataclass
class JulesAccount:
    id: str
    name: str
    api_key: str
    match_owners: List[str] = field(default_factory=list)
    default_branch: str = "main"

class JulesMultiAccountManager:
    """Manages credentials and intelligent routing across multiple Jules/GitHub accounts."""

    def __init__(self):
        self.accounts: Dict[str, JulesAccount] = {}
        self.default_account_id: str = "thedoctormes"
        self._load_from_env()

    def _load_from_env(self):
        # 1. Load thedoctormes-hue account
        key_doc = os.getenv("JULES_API_KEY_DOCTORMES") or os.getenv("JULES_API_KEY") or ""
        if key_doc:
            self.register_account(
                JulesAccount(
                    id="thedoctormes",
                    name="DoctorM & Ai (thedoctormes-hue)",
                    api_key=key_doc,
                    match_owners=["thedoctormes-hue", "thedoctormes", "gyxer513"],
                    default_branch="main"
                )
            )

        # 2. Load TheNovaNodes account
        key_nova = os.getenv("JULES_API_KEY_NOVANODES") or ""
        if key_nova:
            self.register_account(
                JulesAccount(
                    id="thenovanodes",
                    name="TheNovaNodes Organization",
                    api_key=key_nova,
                    match_owners=["thenovanodes", "novanodes"],
                    default_branch="main"
                )
            )

    def register_account(self, account: JulesAccount):
        self.accounts[account.id.lower()] = account
        logger.info(f"Registered Jules account: {account.id} ({account.name}) for owners {account.match_owners}")

    def get_account(self, account_id: str) -> Optional[JulesAccount]:
        return self.accounts.get(account_id.lower())

    def list_accounts(self) -> List[Dict[str, any]]:
        return [
            {
                "id": acc.id,
                "name": acc.name,
                "match_owners": acc.match_owners,
                "has_api_key": bool(acc.api_key),
                "key_preview": f"{acc.api_key[:6]}...{acc.api_key[-4:]}" if acc.api_key else "NONE"
            }
            for acc in self.accounts.values()
        ]

    def resolve_account(self, source_name: Optional[str] = None, explicit_account_id: Optional[str] = None) -> JulesAccount:
        """
        Intelligently resolves the proper JulesAccount instance:
        1. If explicit_account_id is provided, use it.
        2. If source_name is provided (e.g. 'sources/github/TheNovaNodes/nextcloud-mcp-control' or 'TheNovaNodes/repo'),
           match against registered match_owners.
        3. Fallback to default registered account.
        """
        if explicit_account_id:
            acc = self.get_account(explicit_account_id)
            if acc and acc.api_key:
                return acc
            raise ValueError(f"Specified account '{explicit_account_id}' not found or has no API key configured.")

        if source_name:
            # Extract owner from sources/github/owner/repo or owner/repo
            cleaned = source_name.replace("sources/github/", "")
            parts = cleaned.split("/")
            if parts:
                owner = parts[0].strip().lower()
                for acc in self.accounts.values():
                    if any(m.lower() == owner for m in acc.match_owners):
                        if acc.api_key:
                            return acc

        # Fallback to default
        if self.default_account_id in self.accounts and self.accounts[self.default_account_id].api_key:
            return self.accounts[self.default_account_id]

        # Any active account
        for acc in self.accounts.values():
            if acc.api_key:
                return acc

        raise ValueError("No valid Jules API key configured for any account.")
