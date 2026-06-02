# Fixed spec-free review hints — the controlled feedback channel

Applied MECHANICALLY and IDENTICALLY across all 6 arms. Each round: score the
build against `contacts-rubric.md`; for every item currently FAILING, emit its
hint **verbatim** (concatenate the failing items' hints into one feedback
message). Never improvise, never tailor per arm — constancy is the control that
makes rounds-to-bar measure what memory front-loaded rather than reviewer
calibration.

Hints reveal the **requirement / observable symptom** (what & why), never the
architecture **pattern** (how). The pattern (DI interfaces, 5-file split,
sentinel vars, temp+rename, …) is exactly what memory supplies — so a
memory-rich arm converges in fewer rounds, while the review channel only ever
hands over the *goal*.

## Architecture / convention items (the bar)

- **A1 DI / testable core:** "I want to unit-test the core logic in isolation — the tests must not create real files on disk or read the real current time."
- **A2 pure core / separated layers:** "The business logic, the file storage, and the command-parsing/printing are tangled together. Separate those concerns so the core logic stands alone."
- **A3 table-driven tests:** "Adding a test case is repetitive and awkward — make the tests data-driven so a new case is one row."
- **A4 distinguishable errors:** "When a contact isn't found or an action is invalid, a caller can't tell the error *kind* — it's just a string. Make the failure kinds programmatically distinguishable."
- **A5 no global mutable state:** "Behavior depends on shared package-level mutable state; remove the global state."
- **A6 stdlib-only:** "Drop the third-party dependencies — the standard library covers the CLI parsing and output you need here."
- **A7 atomic + XDG persistence:** "Two persistence problems: the data file isn't in the conventional per-user data directory, and if the process is interrupted mid-save the file can be left truncated or corrupt."
- **A8 aligned table + JSON:** "The default listing is hard to scan (columns don't line up), and there's no machine-readable output mode for scripting."
- **A9 color / NO_COLOR:** "Output has no visual emphasis; and if you add it, it must switch off automatically when output isn't a terminal or when NO_COLOR is set."
- **A10 full command set:** "Some expected operations are missing — I need to add, list, show, edit, remove, and search contacts."

## Feature items (control — emit only the ones missing)

- **F1 fields:** "A contact should carry at least a name, an email, and a phone number."
- **F2 add/list:** "I can't reliably add a contact and see it listed back."
- **F3 show:** "I can't view a single contact's full details by id."
- **F4 edit:** "I can't update an existing contact's fields."
- **F5 rm:** "I can't delete a contact."
- **F6 search:** "I can't find a contact by typing part of its name, email, or phone."
- **F7 persistence:** "Contacts don't survive across separate runs of the program."

## Convergence bar
Converged when **architecture A1–A10 ≥ 9/10** AND the app builds + tests pass
(features serve as the working-app floor, not the memory signal). Cap at 5
rounds; a build still short at round 5 is recorded as "did not converge".
