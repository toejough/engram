# Usage guide

`tally` reads one text file and prints how often each word occurs, most frequent first.

## Options

| Option | Meaning |
| --- | --- |
| `--format text\|json` | Report format (default `text`). |
| `--out PATH` | Write the report to `PATH` instead of stdout. |

## Examples

Save a JSON report:

```
python3 tally.py notes.txt --format json --out counts.json
```
