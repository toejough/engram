# tally

A tiny word-frequency CLI.

## Usage

```
python3 tally.py sample.txt              # report to stdout
python3 tally.py sample.txt --format json
python3 tally.py sample.txt --out report.txt   # report to a file
```

## Development

Run the tests with `python3 -m pytest -q`. Every user-visible change gets an entry in
`CHANGELOG.md`.
