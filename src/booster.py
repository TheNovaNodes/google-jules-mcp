"""
Context Booster & Prompt Engineering Engine for Google Jules.
Transforms short/plain user tasks into structured, high-leverage engineering mandates
that maximize the reasoning and code-generation capabilities of Google Jules.
"""

from typing import Optional, List

SYSTEM_PROMPT_TEMPLATE = """\
<audit_mandate>
ROLE & CONTEXT:
You are an expert autonomous software engineer working within the DoctorM & Ai / TheNovaNodes Ecosystem.
You have full access to execute commands, read/write files, run tests, and create high-quality Pull Requests.

TASK OBJECTIVE:
{user_prompt}
</audit_mandate>

{architecture_section}

<verification_contract>
CRITICAL OPERATIONAL RULES & ACCEPTANCE CRITERIA:
1. CODE QUALITY:
   - Write clean, maintainable, modular, and idiomatic code adhering to PEP 8 / TypeScript standard guidelines.
   - All documentation, docstrings, and comments MUST be written in high-quality, professional English.
   - Maintain full backward compatibility unless a breaking change is explicitly requested.

2. STRUCTURE & COMPLIANCE:
   - Ensure standard repository files (README.md, AGENTS.md, CONTRIBUTING.md) are present and up-to-date.
   - Clearly state the repository's role in the DoctorM & Ai / TheNovaNodes and Antigravity Agent Ecosystem.

3. RIGOROUS VERIFICATION (MANDATORY):
   - ALWAYS run the test suite (e.g., `pytest`, `npm test`, `cargo test`) before declaring task completion.
   - Add unit/regression tests for every new feature or bugfix.
   - Zero test failures allowed. Verify that all linters and type checkers pass without errors.

4. COMMIT & PULL REQUEST:
   - Follow Conventional Commits format: `feat(scope): ...`, `fix(scope): ...`, `docs(scope): ...`.
   - Provide a clear, detailed PR description summarizing: Problem Statement, Root Cause, Solution Details, and Verification Results.
</verification_contract>
"""

def boost_prompt(
    user_prompt: str,
    architecture_context: Optional[str] = None,
    target_files: Optional[List[str]] = None,
    test_command: Optional[str] = None
) -> str:
    """
    Synthesizes a master prompt injecting verification contracts, file anchors, and architectural guidelines.
    """
    arch_parts = []
    if target_files:
        files_str = "\n".join(f"- `{f}`" for f in target_files)
        arch_parts.append(f"<file_anchors>\nPRIMARY TARGET FILES:\n{files_str}\n</file_anchors>")

    if architecture_context:
        arch_parts.append(f"<architectural_context>\n{architecture_context.strip()}\n</architectural_context>")

    if test_command:
        arch_parts.append(f"<test_execution_command>\nRun this exact test command: `{test_command}`\n</test_execution_command>")

    arch_section = "\n\n".join(arch_parts)

    return SYSTEM_PROMPT_TEMPLATE.format(
        user_prompt=user_prompt.strip(),
        architecture_section=arch_section
    )
