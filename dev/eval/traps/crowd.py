"""Real-vault variant generator for the crowded-vault capability eval.

Reads the real agent-memory vault READ-ONLY and emits deterministic "crowd" variants — re-slugged
copies of real notes (no LLM paraphrase) with their wikilinks re-pointed to sibling variants — so a
load-bearing note must compete on cosine against a realistic, link-connected crowd. Seeding is
ALWAYS to a temp dir; `seed_into` refuses to write the real vault.

Two seed paths:
  * `seed_into`        — vault axes (C3/C4i/C6): `engram learn fact|feedback` per variant.
  * `seed_into_chunks` — recency axis (C5): `engram ingest --markdown` per variant chunk.

Determinism: `make_variants` uses `random.Random(seed)` with the fixed SEED everywhere.
"""
import math
import os
import random
import re
import shutil
import subprocess
import tempfile

SEED = 7

DEFAULT_VOCAB_FRAC = 0.3   # bias strength when vocab_terms is given but vocab_frac is left at 0.0


def _variant_slug(luhmann_or_src, i):
    """Engram-valid crowd-variant slug for source identifier `luhmann_or_src` at index `i`.

    Real vault notes can carry dotted/uppercase luhmann values (e.g.
    `39.2026-06-17.Integration-Test`), so the raw `crowd-<luhmann>-<i>` slug violates engram's
    `--slug [a-z0-9-]+` rule and a real `engram learn` rejects it. Sanitize to lowercase and collapse
    every other run of disallowed characters to a single hyphen. Used for BOTH the variant slug and
    its re-pointed link targets so links still match sibling slugs after sanitization."""
    return re.sub(r"[^a-z0-9-]+", "-", f"crowd-{luhmann_or_src}-{i}".lower()).strip("-")


def real_vault():
    """Resolve the real vault path: ENGRAM_VAULT_PATH, else $XDG_DATA_HOME/engram/vault, else
    $HOME/.local/share/engram/vault (matches the Go default in internal/cli/targets.go)."""
    explicit = os.environ.get("ENGRAM_VAULT_PATH")
    if explicit:
        return explicit
    xdg = os.environ.get("XDG_DATA_HOME")
    if xdg:
        return os.path.join(xdg, "engram", "vault")
    return os.path.join(os.path.expanduser("~"), ".local", "share", "engram", "vault")


def _parse_note(text, fname):
    """Parse one vault note's YAML frontmatter + Related-to wikilinks. Returns the variant-source
    dict shape (slug=basename, luhmann, type, situation, fields, links) or None if no frontmatter."""
    basename = fname[:-3] if fname.endswith(".md") else fname
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return None
    frontmatter = {}
    cursor = 1
    while cursor < len(lines) and lines[cursor].strip() != "---":
        line = lines[cursor]
        if ":" in line and not line.startswith((" ", "\t")):
            key, _, value = line.partition(":")
            frontmatter[key.strip()] = value.strip().strip('"').strip("'")
        cursor += 1
    body = "\n".join(lines[cursor + 1:])

    note_type = frontmatter.get("type", "fact")
    if note_type == "feedback":
        fields = {"behavior": frontmatter.get("behavior", ""),
                  "impact": frontmatter.get("impact", ""),
                  "action": frontmatter.get("action", "")}
    else:
        fields = {"subject": frontmatter.get("subject", ""),
                  "predicate": frontmatter.get("predicate", ""),
                  "object": frontmatter.get("object", "")}

    related = body.split("Related to:", 1)
    links = re.findall(r"\[\[([^\]]+)\]\]", related[1]) if len(related) > 1 else []

    return {"slug": basename, "luhmann": frontmatter.get("luhmann", ""),
            "type": note_type, "situation": frontmatter.get("situation", ""),
            "fields": fields, "links": links}


def load_real_notes(vault_path):
    """Read every *.md note in vault_path READ-ONLY and parse it into a source-note dict."""
    notes = []
    for fname in sorted(os.listdir(vault_path)):
        if not fname.endswith(".md"):
            continue
        with open(os.path.join(vault_path, fname), encoding="utf-8", errors="ignore") as handle:
            note = _parse_note(handle.read(), fname)
        if note is not None:
            notes.append(note)
    return notes


def _vocab_matches(note, terms):
    """True if any lowercase term appears in the note's identifying field text."""
    fields = note.get("fields", {})
    if note.get("type") == "feedback":
        haystack = (fields.get("behavior", "") + " " + fields.get("action", "")).lower()
    else:
        haystack = (fields.get("subject", "") + " " + fields.get("object", "")).lower()
    return any(term in haystack for term in terms)


def _ordered_sources(notes, n, rng, vocab_terms, vocab_frac):
    """Build an ordered, shuffled source list of length n, biased toward vocab-matching notes."""
    if vocab_terms:
        terms = [t.lower() for t in vocab_terms]
        match = [note for note in notes if _vocab_matches(note, terms)]
        rest = [note for note in notes if not _vocab_matches(note, terms)]
    else:
        match, rest = [], list(notes)

    n_match = min(math.ceil(vocab_frac * n), n) if (vocab_terms and match) else 0
    sources = [match[i % len(match)] for i in range(n_match)]

    remainder_pool = rest if rest else list(notes)
    for i in range(n - n_match):
        sources.append(remainder_pool[i % len(remainder_pool)])

    rng.shuffle(sources)
    return sources


def make_variants(notes, n, seed=SEED, vocab_terms=(), vocab_frac=0.0, recency_frac=0.0):
    """Deterministically emit n crowd variants (re-slugged real notes) with links re-pointed to
    sibling variants. Pure — no I/O. Same seed => identical output."""
    rng = random.Random(seed)
    if vocab_terms and vocab_frac == 0.0:
        vocab_frac = DEFAULT_VOCAB_FRAC   # an explicit topic but no strength => bias by default
    sources = _ordered_sources(notes, n, rng, vocab_terms, vocab_frac)

    variants = []
    for i, source in enumerate(sources):
        variants.append({
            "slug": _variant_slug(source["luhmann"], i),
            "src_slug": source["slug"],
            "type": source.get("type", "fact"),
            "situation": source.get("situation", ""),
            "fields": dict(source.get("fields", {})),
            "links": [],
            "newer": i < recency_frac * n,
        })

    # Link re-pointing: a source's link -> target_basename becomes a link to the target's variant
    # at the same index i, else wraps within the target's variants, else is dropped.
    luhmann_by_slug = {note["slug"]: note["luhmann"] for note in notes}
    variants_by_src = {}
    for variant in variants:
        variants_by_src.setdefault(variant["src_slug"], []).append(variant)
    slug_set = {variant["slug"] for variant in variants}

    for i, (source, variant) in enumerate(zip(sources, variants)):
        repointed = []
        for target_basename in source.get("links", []):
            candidates = variants_by_src.get(target_basename)
            if not candidates:
                continue                                   # target not in the crowd -> drop
            same_index_slug = _variant_slug(luhmann_by_slug[target_basename], i)
            if same_index_slug in slug_set:
                repointed.append(same_index_slug)
            else:
                repointed.append(candidates[i % len(candidates)]["slug"])
        variant["links"] = repointed

    return variants


def _field_args(variant):
    """CLI field flags for the variant's type."""
    fields = variant["fields"]
    if variant["type"] == "feedback":
        return ["--behavior", fields.get("behavior", ""),
                "--impact", fields.get("impact", ""),
                "--action", fields.get("action", "")]
    return ["--subject", fields.get("subject", ""),
            "--predicate", fields.get("predicate", ""),
            "--object", fields.get("object", "")]


def seed_into(vault_path, variants):
    """Seed crowd variant NOTES into vault_path via `engram learn`. Refuses the real vault.

    Note: `--position top` is used (not the plan's `sibling`): `engram learn --position sibling`
    requires a `--target` Luhmann ID, which a free-standing crowd variant does not have. `top`
    gives each variant a fresh top-level ID — the same working pattern seed_c3 uses."""
    if os.path.realpath(vault_path) == os.path.realpath(real_vault()):
        raise RuntimeError(f"refusing to seed into the real vault: {vault_path}")
    os.makedirs(vault_path, exist_ok=True)
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault_path
    for variant in variants:
        cmd = ["engram", "learn", variant["type"],
               "--slug", variant["slug"], "--position", "top",
               "--source", "crowd", "--situation", variant["situation"]] + _field_args(variant)
        for link in variant["links"]:
            cmd += ["--relation", f"{link}|crowd"]
        result = subprocess.run(cmd, env=env, capture_output=True, text=True)
        if result.returncode != 0:
            raise RuntimeError(
                f"engram learn {variant['type']} failed for {variant['slug']!r} "
                f"(exit {result.returncode}): {result.stderr.strip()}")


def seed_into_chunks(chunks_dir, variants):
    """Seed crowd variant CHUNKS into chunks_dir via `engram ingest --markdown` (C5 only). The
    caller seeds these BEFORE R so R stays the newest chunk.

    Ingest order is by the variant `newer` flag ascending (False before True) so `newer=True` chunks
    land most-recently within the crowd — the only place `recency_frac` has a real effect, since
    chunk recency is by ingest time (`engram learn` cannot stamp a created-date on a vault note)."""
    os.makedirs(chunks_dir, exist_ok=True)
    src_dir = tempfile.mkdtemp(prefix="crowd-chunks-src-")
    try:
        for variant in sorted(variants, key=lambda v: v.get("newer", False)):
            fields = variant["fields"]
            if variant["type"] == "feedback":
                field_text = " ".join([fields.get("behavior", ""), fields.get("impact", ""),
                                        fields.get("action", "")])
            else:
                field_text = " ".join([fields.get("subject", ""), fields.get("predicate", ""),
                                        fields.get("object", "")])
            body = f"# {variant['slug']}\n\n{variant['situation']}\n\n{field_text}\n"
            path = os.path.join(src_dir, f"{variant['slug']}.md")
            with open(path, "w", encoding="utf-8") as handle:
                handle.write(body)
            result = subprocess.run(
                ["engram", "ingest", "--markdown", path, "--chunks-dir", chunks_dir],
                capture_output=True, text=True)
            if result.returncode != 0:
                raise RuntimeError(
                    f"engram ingest failed for {variant['slug']!r} "
                    f"(exit {result.returncode}): {result.stderr.strip()}")
    finally:
        shutil.rmtree(src_dir, ignore_errors=True)
