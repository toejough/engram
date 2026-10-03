## ADDED Requirements

### Requirement: The existence check's `engram query` invocation SHALL be safe against any phrase/text value, and SHALL NOT silently swallow a real failure

The point-in-time existence check (`dev/eval/audit/audit_moments.py::_run_engram_query_at_moment`, or any function building an `engram query` CLI invocation from auditor-generated phrase/text strings) SHALL pass each phrase/text value in a form `engram`'s CLI flag parser cannot mistake for a new flag, regardless of the value's own content (including a value that itself starts with `--`). If the invocation still fails (non-zero exit), the function's returned error information SHALL include the tool's actual error output — on whichever stream it was written — not merely whichever stream happened to be empty. A failed existence check SHALL be diagnosable from its own recorded error field alone, without needing to re-run the command by hand.

#### Scenario: A phrase starting with `--` does not break the invocation

- **WHEN** the existence check is given a search phrase whose text begins with `--` (e.g. `"--text verbatim scenario dropped from modified requirement"`)
- **THEN** `engram query` runs successfully against that phrase and the existence check's result reflects a real query outcome, not a CLI flag-parsing error

#### Scenario: A real `engram query` failure's error text is preserved

- **WHEN** `engram query` exits non-zero and writes its error message to stdout (not stderr)
- **THEN** the existence check's returned `error` field contains that error message, never an empty or uninformative string
