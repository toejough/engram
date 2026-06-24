#!/usr/bin/env python3
"""Recency channel probe: parse engram query payload for recent-provenance items (C5 axis)."""


def _parse_yaml_items(text):
    """Minimal YAML parser for engram query payloads: extracts items list."""
    import json as _json
    # Try JSON first (also valid)
    try:
        return _json.loads(text)
    except Exception:
        pass
    # Parse simple YAML: lines starting with "  - " begin a new item; "    key: value" are fields.
    data = {"items": []}
    current_item = None
    in_items = False
    current_list_key = None
    for line in text.splitlines():
        stripped = line.rstrip()
        if not stripped or stripped.startswith("#"):
            continue
        if stripped.startswith("items:"):
            in_items = True
            continue
        if not in_items:
            continue
        # New item
        if stripped.startswith("  - ") or stripped.startswith("- "):
            if current_item is not None:
                data["items"].append(current_item)
            current_item = {}
            current_list_key = None
            # Handle "  - key: value" on same line
            rest = stripped.lstrip("- ").strip()
            if ":" in rest:
                k, _, v = rest.partition(":")
                current_item[k.strip()] = v.strip().strip('"')
        elif current_item is not None and ":" in stripped:
            indent = len(stripped) - len(stripped.lstrip())
            rest = stripped.strip()
            k, _, v = rest.partition(":")
            k = k.strip()
            v = v.strip().strip('"')
            if v == "":
                # key with no value — likely a list follows
                current_list_key = k
                current_item[k] = []
            elif v.startswith("[") and v.endswith("]"):
                # inline list
                inner = v[1:-1]
                current_item[k] = [x.strip().strip('"') for x in inner.split(",") if x.strip()]
                current_list_key = None
            else:
                current_item[k] = v
                current_list_key = None
        elif current_item is not None and stripped.startswith("      - ") or (
                current_item is not None and stripped.startswith("    - ")):
            val = stripped.lstrip("- ").strip().strip('"')
            if current_list_key and isinstance(current_item.get(current_list_key), list):
                current_item[current_list_key].append(val)
    if current_item is not None:
        data["items"].append(current_item)
    return data


def parse_recent_channel(payload_yaml_str):
    """Return items whose provenances list includes 'recent'."""
    data = _parse_yaml_items(payload_yaml_str)
    items = data.get("items", []) if isinstance(data, dict) else []
    result = []
    for item in items:
        provs = item.get("provenances") or []
        if isinstance(provs, str):
            provs = [provs]
        if "recent" in provs:
            result.append(dict(item, provenance="recent"))
    return result


def recent_channel_surfaced(payload_yaml_str, target_note_basename):
    """True if target_note_basename appears in the recent channel.

    Chunk paths carry a '#anchor' suffix (e.g. 'R-decision.md#Section title'); notes do not.
    Strip the anchor before matching so a chunk source matches on its filename, not the whole
    path-with-anchor."""
    items = parse_recent_channel(payload_yaml_str)
    for item in items:
        path = item.get("path", "")
        base = path.split("/")[-1]
        base_no_anchor = base.split("#", 1)[0]          # drop chunk #anchor
        base_no_ext = base_no_anchor[:-3] if base_no_anchor.endswith(".md") else base_no_anchor
        if target_note_basename in (path, base, base_no_anchor, base_no_ext):
            return True
    return False


def score_recency_hit(recall_payload_str, target_basename):
    """Return recent channel item count and whether target was surfaced."""
    items = parse_recent_channel(recall_payload_str)
    return {
        "recent_channel_items": len(items),
        "target_surfaced": recent_channel_surfaced(recall_payload_str, target_basename),
    }
