"""
Mechanical JSONL transcript parser for Claude Code sessions.

Extracts structured events (recall calls, dispatch events, learn calls, engram commands)
from main sessions and subagent transcripts (fork/fresh/workflow types).
"""

import json
import os
import re
from pathlib import Path
from typing import Any, Dict, List, Optional


def is_engram_query_directly_executed(command: str) -> bool:
    """
    Check if this bash command directly executes an engram query.

    Returns False (skip counting as engram query) if the command invokes
    a subprocess dispatch (pi -p or claude -p) that CONTAINS the "engram query"
    substring — meaning engram query is being passed as an instruction to a
    child process, not executed directly in this bash call.

    This prevents false-positive detection of engram query calls inside
    quoted prompt strings being passed to subprocess dispatches.

    Heuristic limitation: a command that BOTH dispatches a subprocess AND
    runs a real engram query afterward as a separate step would be a false
    negative (not detected). That's acceptable and documented.
    """
    if "engram query" not in command:
        return False

    # Find positions of subprocess dispatch patterns
    pi_p_pos = command.find("pi -p")
    claude_p_pos = command.find("claude -p")
    engram_query_pos = command.find("engram query")

    # If engram query appears before any subprocess dispatch, it's direct
    if pi_p_pos == -1 and claude_p_pos == -1:
        return True

    # If a subprocess dispatch appears BEFORE engram query, the engram query
    # is inside the subprocess dispatch's quoted prompt string, so skip it
    subprocess_pos = min(
        pi_p_pos if pi_p_pos != -1 else float("inf"),
        claude_p_pos if claude_p_pos != -1 else float("inf"),
    )

    return engram_query_pos < subprocess_pos


def normalize_tool_result_content(content: Any) -> str:
    """
    Normalize tool_result content to a plain string.

    Content can be either:
    - A bare string
    - A list of blocks like [{"type": "text", "text": "..."}]
    """
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts = []
        for block in content:
            if isinstance(block, dict) and block.get("type") == "text":
                parts.append(block.get("text", ""))
        return "".join(parts)
    return str(content)


def parse_engram_query_result(result_text: str) -> List[Dict[str, Any]]:
    """
    Parse the YAML-like output from an engram query command.

    Extracts items with path, kind, score, and provenances.
    Returns a list of {path, kind, score, source, rank}.

    Handles two formats:
    1. Full format with version header and structured fields (kind, score, provenances)
    2. Bare-list legacy format with just "- path:" lines at the top level (no items: section)
       where kind/score/source are not present
    """
    items = []
    lines = result_text.split("\n")
    in_items_section = False
    current_rank = 0

    # Check if this is a bare-list legacy format (starts with "  - path:" or "- path:" with no "items:" header)
    stripped_text = result_text.strip()
    is_bare_list = (stripped_text and not stripped_text.startswith("version:") and
                    ("- path:" in stripped_text) and "items:" not in result_text)

    i = 0
    while i < len(lines):
        line = lines[i]

        # Detect start of items section
        if line.strip().startswith("items:"):
            in_items_section = True
            i += 1
            continue

        # Detect end of items section (clusters start)
        if in_items_section and line.strip().startswith("clusters:"):
            break

        # Parse item entries (they start with "  - path:" or "- path:")
        # Either inside an items section OR as bare-list format
        if (in_items_section or is_bare_list) and (line.startswith("  - path:") or line.startswith("- path:")):
            current_rank += 1
            item = {"rank": current_rank}

            # Parse path
            path_match = re.match(r'\s*-\s+path:\s+(.+?)$', line)
            if path_match:
                item["path"] = path_match.group(1).strip()

            # Look ahead for other fields (only in full format, not bare-list)
            if in_items_section:
                i += 1
                while i < len(lines):
                    line = lines[i]

                    # Stop at next item or section
                    if (line.startswith("  - path:") or line.startswith("- path:") or
                        line.strip().startswith("clusters:")):
                        i -= 1  # Back up one line so the outer loop processes it
                        break

                    # Parse kind
                    if line.strip().startswith("kind:"):
                        kind_match = re.match(r'\s+kind:\s+(.+?)$', line)
                        if kind_match:
                            item["kind"] = kind_match.group(1).strip()

                    # Parse score
                    if line.strip().startswith("score:"):
                        score_match = re.match(r'\s+score:\s+(.+?)$', line)
                        if score_match:
                            try:
                                item["score"] = float(score_match.group(1).strip())
                            except ValueError:
                                item["score"] = 0

                    # Parse provenances
                    if line.strip().startswith("provenances:"):
                        i += 1
                        provenances = []
                        while i < len(lines):
                            prov_line = lines[i]
                            if prov_line.startswith("      -"):
                                prov_match = re.match(r'\s+-\s+(.+?)$', prov_line)
                                if prov_match:
                                    provenances.append(prov_match.group(1).strip())
                                i += 1
                            else:
                                i -= 1
                                break
                        item["source"] = provenances[0] if provenances else "unknown"
                        i += 1
                        continue

                    # Stop at content block (multiline, starts with "content:")
                    if line.strip().startswith("content:"):
                        # Skip content block; it's not needed for the audit
                        i += 1
                        while i < len(lines):
                            next_line = lines[i]
                            # Content block ends when we hit a line at the same or lower indent
                            # as "kind:", "score:", etc. (2-space indent)
                            if next_line and not next_line.startswith("      "):
                                if (next_line.startswith("  - path:") or next_line.startswith("- path:") or
                                    next_line.strip().startswith("clusters:")):
                                    i -= 1
                                break
                            i += 1
                        i += 1
                        continue

                    i += 1
            else:
                # Bare-list format: no fields to look ahead for, just extract the path
                pass

            if item.get("path"):
                items.append(item)
            i += 1
            continue

        i += 1

    return items


def determine_dispatch_type(transcript_path: str, meta_data: Optional[Dict[str, Any]] = None) -> str:
    """
    Determine dispatch type: "main", "fork", "fresh", or "workflow".

    Rules:
    - If path contains "/subagents/workflows/", it's "workflow"
    - If meta.json exists and has isFork=true or agentType="fork", it's "fork"
    - If path is under /subagents/ (but not workflows), it's "fresh"
    - Otherwise it's "main"
    """
    path = transcript_path.lower()

    # Workflow detection by path
    if "/subagents/workflows/" in path:
        return "workflow"

    # Fork detection by meta.json
    if meta_data:
        if meta_data.get("isFork") is True or meta_data.get("agentType") == "fork":
            return "fork"

    # Fresh detection by path
    if "/subagents/" in path:
        return "fresh"

    # Default to main
    return "main"


def read_meta_json(transcript_path: str) -> Optional[Dict[str, Any]]:
    """
    Read the .meta.json sidecar for a subagent transcript if it exists.
    """
    meta_path = transcript_path.replace(".jsonl", ".meta.json")
    if os.path.exists(meta_path):
        try:
            with open(meta_path) as f:
                return json.load(f)
        except (json.JSONDecodeError, IOError):
            return None
    return None


def detect_transcript_format(path: str) -> str:
    """
    Detect whether a transcript file is Claude Code or Pi format.

    Returns "claude_code" or "pi".

    Detection logic: Read the first JSON line. If it has {"type": "session", "version": 3}, it's Pi.
    Otherwise, it's Claude Code.
    """
    try:
        with open(path) as f:
            first_line = f.readline().strip()
            if not first_line:
                return "claude_code"

            try:
                obj = json.loads(first_line)
                if obj.get("type") == "session" and obj.get("version") == 3:
                    return "pi"
            except json.JSONDecodeError:
                pass
    except IOError:
        pass

    return "claude_code"


def extract_transcript_pi(path: str) -> Dict[str, Any]:
    """
    Extract structured events from a Pi agent transcript JSONL file.

    Pi format uses JSONL with message lines of type "message", containing:
    - role: "user", "assistant", or "toolResult"
    - content: list of blocks for user/assistant, or structured content for toolResult
    """
    result = {
        "transcript_path": path,
        "dispatch_type": "pi_main",
        "recall_calls": [],
        "dispatch_events": [],
        "learn_calls": [],  # Always empty for Pi (engram commands appear as bash calls)
        "engram_commands": [],
        "completion_report": None,
        "classification_mismatches": [],  # Always empty for Pi (no meta.json)
    }

    # Detect if this is a nested subagent run
    # Nested Pi subagent files have paths like: ~/.pi/agent/sessions/<hash>/<subagent-hash>/run-N/session.jsonl
    if "/run-" in path and "/session.jsonl" in path:
        result["dispatch_type"] = "pi_subagent"

    # Track recall call state within Pi's bash commands
    current_recall_call = None
    bash_tool_use_stack = {}  # Maps bash tool_use_id to {location, timestamp, command}
    assistant_lines = []  # Keep track of assistant message lines for completion report

    try:
        with open(path) as f:
            line_num = 0
            for line in f:
                line_num += 1
                line = line.strip()
                if not line:
                    continue

                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue

                event_type = event.get("type")

                # Only process message events
                if event_type != "message":
                    continue

                msg = event.get("message", {})
                role = msg.get("role")
                timestamp = event.get("timestamp")

                # Track assistant messages for completion report
                if role == "assistant":
                    assistant_lines.append((line_num, event))
                    content = msg.get("content", [])

                    for block in content:
                        if not isinstance(block, dict):
                            continue

                        block_type = block.get("type")

                        if block_type == "toolCall":
                            tool_name = block.get("name")
                            tool_id = block.get("id", "")
                            arguments = block.get("arguments", {})

                            # Handle bash tool calls (engram query, learn, amend, activate)
                            if tool_name == "bash":
                                command = arguments.get("command", "")

                                # Check if this is an engram query (for recall calls)
                                # Gap 1 fix: skip if engram query is inside a pi -p or claude -p subprocess dispatch
                                if is_engram_query_directly_executed(command):
                                    # For Pi, we don't have a recall Skill marker, so we create
                                    # a recall call directly from the bash engram query command
                                    # (Unlike Claude Code, recall is not a separate Skill call)
                                    if current_recall_call is not None:
                                        result["recall_calls"].append(current_recall_call)

                                    # Extract phrases from command if possible
                                    query_phrases = []
                                    for match in re.finditer(r'--phrase\s+"([^"]+)"', command):
                                        query_phrases.append(match.group(1))

                                    current_recall_call = {
                                        "location": line_num,
                                        "timestamp": timestamp,
                                        "skill_args": "",  # Pi doesn't have skill args
                                        "query_phrases": query_phrases,
                                        "matched_items": [],
                                        "possibly_truncated": False,  # Gap 2: added field, initialized to False
                                    }

                                    bash_tool_use_stack[tool_id] = {
                                        "location": line_num,
                                        "timestamp": timestamp,
                                        "command": command,
                                    }

                                # Check if this is an engram learn/amend/activate command
                                # Gap 1 fix: also apply to these commands
                                if any(cmd in command for cmd in ["engram learn", "engram amend", "engram activate"]):
                                    # Additional check: only if not inside a subprocess dispatch
                                    is_subprocess_dispatch = (
                                        ("pi -p" in command and command.find("pi -p") < command.find(next((c for c in ["engram learn", "engram amend", "engram activate"] if c in command), "")))
                                        or ("claude -p" in command and command.find("claude -p") < command.find(next((c for c in ["engram learn", "engram amend", "engram activate"] if c in command), "")))
                                    )
                                    if not is_subprocess_dispatch:
                                        cmd_type = None
                                        if "engram learn" in command:
                                            cmd_type = "learn"
                                        elif "engram amend" in command:
                                            cmd_type = "amend"
                                        elif "engram activate" in command:
                                            cmd_type = "activate"

                                        result["engram_commands"].append({
                                            "location": line_num,
                                            "timestamp": timestamp,
                                            "command": cmd_type,
                                            "full_command_text": command,
                                        })

                            # Handle subagent tool calls (dispatch events)
                            elif tool_name == "subagent":
                                tasks = arguments.get("tasks", [])
                                # Extract one dispatch_event per task
                                for task in tasks:
                                    agent = task.get("agent", "")
                                    model = task.get("model", None)
                                    task_text = task.get("task", "")

                                    result["dispatch_events"].append({
                                        "location": line_num,
                                        "timestamp": timestamp,
                                        "tool_use_id": tool_id,
                                        "description": agent,  # Pi doesn't separate description from subagent_type
                                        "subagent_type": agent,
                                        "model": model,
                                        "prompt_text": task_text,
                                    })

                # Handle tool results (correlate with bash tool calls)
                elif role == "toolResult":
                    tool_call_id = msg.get("toolCallId", "")
                    content = msg.get("content", [])

                    # Normalize content
                    result_content = normalize_tool_result_content(content)

                    # Check if this is a result for an engram query bash call
                    if tool_call_id in bash_tool_use_stack and current_recall_call is not None:
                        # Parse the engram query result
                        matched_items = parse_engram_query_result(result_content)

                        # Attach the query result to the current recall call
                        current_recall_call["matched_items"].extend(matched_items)

                        # Gap 2: detect if result is possibly truncated
                        # A result is truncated if it has substantial content but doesn't start with "version:"
                        result_stripped = result_content.strip()
                        if result_stripped and not result_stripped.startswith("version:") and len(result_stripped) > 100:
                            current_recall_call["possibly_truncated"] = True

                        # Clean up the bash tool_use tracking
                        del bash_tool_use_stack[tool_call_id]

                        # Close the recall call after processing its result
                        result["recall_calls"].append(current_recall_call)
                        current_recall_call = None

    except IOError as e:
        raise IOError(f"Error reading transcript {path}: {e}")

    # Close any remaining open recall call at EOF
    if current_recall_call is not None:
        result["recall_calls"].append(current_recall_call)

    # Extract completion report (last assistant message for subagents)
    if result["dispatch_type"] == "pi_subagent" and assistant_lines:
        completion_report = None
        for line_num, event in reversed(assistant_lines):
            message = event.get("message", {})
            content = message.get("content", [])

            # Extract text blocks
            completion_parts = []
            for block in content:
                if isinstance(block, dict):
                    if block.get("type") == "text":
                        completion_parts.append(block.get("text", ""))

            if completion_parts:
                completion_report = "".join(completion_parts)
                break

        if completion_report is not None:
            result["completion_report"] = completion_report

    return result


def extract_transcript(path: str) -> Dict[str, Any]:
    """
    Extract structured events from a transcript JSONL file.

    Automatically detects format (Claude Code or Pi) and delegates to the appropriate parser.

    Args:
        path: Path to the transcript JSONL file

    Returns:
        A dict with keys:
        - transcript_path: the input path
        - dispatch_type: "main", "fork", "fresh", "workflow" (Claude Code), or "pi_main", "pi_subagent" (Pi)
        - recall_calls: list of recall events
        - dispatch_events: list of Agent calls
        - learn_calls: list of learn/write-memory calls
        - engram_commands: list of engram commands
        - completion_report: final assistant text or None
        - classification_mismatches: list of classification mismatches
    """
    # Detect format
    fmt = detect_transcript_format(path)
    if fmt == "pi":
        return extract_transcript_pi(path)

    # Claude Code format (existing logic)
    result = {
        "transcript_path": path,
        "dispatch_type": "main",
        "recall_calls": [],
        "dispatch_events": [],
        "learn_calls": [],
        "engram_commands": [],
        "completion_report": None,
        "classification_mismatches": [],
    }

    # Read meta.json for classification
    meta_data = read_meta_json(path)
    dispatch_type = determine_dispatch_type(path, meta_data)
    result["dispatch_type"] = dispatch_type

    # Track recall call state: when a Skill(recall) is encountered, we open a current_recall_call.
    # Subsequent Bash tool_uses with "engram query" attach to this recall call.
    # The recall call closes when we hit the next Skill(recall) or EOF.
    current_recall_call = None  # Dict with location, timestamp, skill_args, query_phrases, matched_items
    bash_tool_use_stack = {}  # Maps bash tool_use_id to (location, timestamp) for engram query tracking
    assistant_lines = []  # Keep track of assistant message lines for completion report

    # Bug 1: Only apply bare-bash recall logic for subagents (fresh/workflow/fork), not main.
    # Main sessions have internal skill queries (e.g., route's evidence lookup) that should not
    # be captured as recall calls.
    allow_bare_bash_recall = dispatch_type != "main"

    try:
        with open(path) as f:
            line_num = 0
            for line in f:
                line_num += 1
                line = line.strip()
                if not line:
                    continue

                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    # Skip malformed lines
                    continue

                event_type = event.get("type")

                # Track assistant messages for completion report
                if event_type == "assistant":
                    assistant_lines.append((line_num, event))

                    # Extract Skill and Agent tool_use blocks
                    message = event.get("message", {})
                    content = message.get("content", [])

                    for block in content:
                        if not isinstance(block, dict):
                            continue

                        block_type = block.get("type")

                        # Track Skill calls
                        if block_type == "tool_use" and block.get("name") == "Skill":
                            tool_use_id = block.get("id", "")
                            skill_name = block.get("input", {}).get("skill", "")
                            skill_args = block.get("input", {}).get("args", "")

                            if skill_name == "recall":
                                # Close any open recall call before opening a new one
                                if current_recall_call is not None:
                                    result["recall_calls"].append(current_recall_call)

                                # Open a new recall call
                                current_recall_call = {
                                    "location": line_num,
                                    "timestamp": event.get("timestamp"),
                                    "skill_args": skill_args,
                                    "query_phrases": [],
                                    "matched_items": [],
                                    "possibly_truncated": False,  # Gap 2: added field, initialized to False
                                }
                            else:
                                # Any skill OTHER than recall closes an open recall window,
                                # EXCEPT write-memory: recall's own documented procedure
                                # legitimately nests a write-memory call mid-execution
                                # (Step 2.5C coverage write, Step 4 persist synthesis) and
                                # then continues the SAME recall invocation afterward
                                # (e.g. a Step 3.5 re-entry query). Without this exception,
                                # other skills' own "engram query" Bash calls (e.g. route's
                                # dispatch-evidence tally query) that happen to fall inside
                                # a still-open recall window would get wrongly merged into
                                # that recall call's matched_items/query_phrases.
                                if skill_name != "write-memory" and current_recall_call is not None:
                                    result["recall_calls"].append(current_recall_call)
                                    current_recall_call = None

                                if skill_name in ("learn", "write-memory"):
                                    result["learn_calls"].append({
                                        "location": line_num,
                                        "timestamp": event.get("timestamp"),
                                        "skill": skill_name,
                                        "skill_args": skill_args,
                                    })

                        # Track Agent calls (dispatch events)
                        elif block_type == "tool_use" and block.get("name") == "Agent":
                            tool_use_id = block.get("id", "")
                            agent_input = block.get("input", {})
                            agent_subagent_type = agent_input.get("subagent_type", "")
                            agent_model = agent_input.get("model", None)
                            agent_description = agent_input.get("description", "")
                            agent_prompt = agent_input.get("prompt", "")

                            # Check for classification mismatch
                            if dispatch_type != "main" and meta_data:
                                meta_agent_type = meta_data.get("agentType", "")
                                if (meta_agent_type == "fork" and agent_subagent_type != "fork" and
                                    agent_subagent_type != ""):
                                    result["classification_mismatches"].append({
                                        "tool_use_id": tool_use_id,
                                        "parent_declared": agent_subagent_type,
                                        "meta_declared": meta_agent_type,
                                    })

                            result["dispatch_events"].append({
                                "location": line_num,
                                "timestamp": event.get("timestamp"),
                                "tool_use_id": tool_use_id,
                                "description": agent_description,
                                "subagent_type": agent_subagent_type,
                                "model": agent_model,
                                "prompt_text": agent_prompt,
                            })

                        # Track Bash commands for engram operations and recall queries
                        elif block_type == "tool_use" and block.get("name") == "Bash":
                            tool_use_id = block.get("id", "")
                            command = block.get("input", {}).get("command", "")

                            # Check if this is an engram query (for recall calls)
                            # Bug 1 fix: also recognize bare-Bash engram query calls (without preceding Skill(recall))
                            # Only in subagents (fresh/workflow/fork), not in main sessions where internal skills
                            # (e.g., route) run their own engram queries
                            if is_engram_query_directly_executed(command):
                                if current_recall_call is None and allow_bare_bash_recall:
                                    # Bare-Bash engram query in a subagent: open a new recall window
                                    # (no Skill(recall) wrapper preceded it)
                                    current_recall_call = {
                                        "location": line_num,
                                        "timestamp": event.get("timestamp"),
                                        "skill_args": "",  # Empty skill_args marks this as bare-bash
                                        "query_phrases": [],
                                        "matched_items": [],
                                        "possibly_truncated": False,
                                    }

                                # Track this bash tool_use so we can correlate it with its result
                                # If skill_args is empty, this is a bare-Bash recall (no Skill(recall) wrapper)
                                if current_recall_call is not None:
                                    bash_tool_use_stack[tool_use_id] = {
                                        "location": line_num,
                                        "timestamp": event.get("timestamp"),
                                        "command": command,
                                        "bare_bash_recall": current_recall_call.get("skill_args", "") == "",
                                    }

                            # Check if this is an engram learn/amend/activate command
                            # Gap 1 fix: also apply to these commands
                            if any(cmd in command for cmd in ["engram learn", "engram amend", "engram activate"]):
                                # Additional check: only if not inside a subprocess dispatch
                                is_subprocess_dispatch = (
                                    ("pi -p" in command and command.find("pi -p") < command.find(next((c for c in ["engram learn", "engram amend", "engram activate"] if c in command), "")))
                                    or ("claude -p" in command and command.find("claude -p") < command.find(next((c for c in ["engram learn", "engram amend", "engram activate"] if c in command), "")))
                                )
                                if not is_subprocess_dispatch:
                                    # Determine which engram command
                                    cmd_type = None
                                    if "engram learn" in command:
                                        cmd_type = "learn"
                                    elif "engram amend" in command:
                                        cmd_type = "amend"
                                    elif "engram activate" in command:
                                        cmd_type = "activate"

                                    result["engram_commands"].append({
                                        "location": line_num,
                                        "timestamp": event.get("timestamp"),
                                        "command": cmd_type,
                                        "full_command_text": command,
                                    })

                # Extract tool results (correlate with recall calls)
                elif event_type == "user":
                    message = event.get("message", {})
                    content = message.get("content")

                    # Normalize content to handle both string and list forms
                    if isinstance(content, str):
                        # content is a plain string
                        pass
                    elif isinstance(content, list):
                        # content is a list of blocks
                        for block in content:
                            if isinstance(block, dict) and block.get("type") == "tool_result":
                                tool_use_id = block.get("tool_use_id", "")
                                result_content = normalize_tool_result_content(block.get("content", ""))

                                # Check if this is a result for an engram query bash call
                                if tool_use_id in bash_tool_use_stack and current_recall_call is not None:
                                    # Parse the engram query result
                                    query_phrases = []
                                    matched_items = []

                                    # Try to parse phrases section
                                    if "phrases:" in result_content:
                                        phrases_match = re.search(
                                            r'phrases:\s*\n((?:\s+-\s+.+\n)*)', result_content
                                        )
                                        if phrases_match:
                                            phrase_lines = phrases_match.group(1).split("\n")
                                            for phrase_line in phrase_lines:
                                                phrase_match = re.match(r'\s+-\s+(.+?)$', phrase_line)
                                                if phrase_match:
                                                    query_phrases.append(phrase_match.group(1).strip())

                                    # Parse items
                                    matched_items = parse_engram_query_result(result_content)

                                    # Attach the query result to the current recall call
                                    current_recall_call["query_phrases"].extend(query_phrases)
                                    current_recall_call["matched_items"].extend(matched_items)

                                    # Gap 2: detect if result is possibly truncated
                                    # A result is truncated if it has substantial content but doesn't start with "version:"
                                    result_stripped = result_content.strip()
                                    if result_stripped and not result_stripped.startswith("version:") and len(result_stripped) > 100:
                                        current_recall_call["possibly_truncated"] = True

                                    # Bug 1: Check if this was a bare-Bash recall call
                                    is_bare_bash = bash_tool_use_stack[tool_use_id].get("bare_bash_recall", False)
                                    del bash_tool_use_stack[tool_use_id]

                                    # If this was a bare-Bash call, close it immediately
                                    if is_bare_bash:
                                        result["recall_calls"].append(current_recall_call)
                                        current_recall_call = None

    except IOError as e:
        raise IOError(f"Error reading transcript {path}: {e}")

    # Close any remaining open recall call at EOF
    if current_recall_call is not None:
        result["recall_calls"].append(current_recall_call)

    # Extract completion report (last assistant message for subagents)
    # Iterate backwards through assistant_lines to find the last message with text blocks
    if dispatch_type != "main" and assistant_lines:
        completion_report = None
        for line_num, event in reversed(assistant_lines):
            message = event.get("message", {})
            content = message.get("content", [])

            # Extract text blocks
            completion_parts = []
            for block in content:
                if isinstance(block, dict):
                    if block.get("type") == "text":
                        completion_parts.append(block.get("text", ""))

            if completion_parts:
                completion_report = "".join(completion_parts)
                break

        if completion_report is not None:
            result["completion_report"] = completion_report

    return result


if __name__ == "__main__":
    import sys

    if len(sys.argv) < 2:
        print("Usage: python extract.py <transcript_path>")
        sys.exit(1)

    transcript_path = sys.argv[1]
    result = extract_transcript(transcript_path)
    print(json.dumps(result, indent=2))
