## 1. Verify evidence is current

- [x] 1.1 Re-run `engram count --group-by tags --filter tags=work-kind/go-package-implementation --filter tags=tier/cheap` and the `tier/mid` equivalent (with and without `--filter tags=outcome/pass`) to confirm the 8/12 cheap / 10/10 mid tally still holds (or capture the current numbers if it has moved). Confirmed unchanged: cheap 8/12 pass, mid 10/10 pass (2026-08-09).

## 2. RED — baseline test

- [ ] 2.1 Using `superpowers:writing-skills`, write a baseline test/fixture demonstrating that the current Cold-start priors table under-routes `work-kind/go-package-implementation` to the cheap tier absent recall (the gap this change closes).

## 3. GREEN — priors table edit

- [ ] 3.1 Add a `work-kind/go-package-implementation` row to the Cold-start priors table in `agent-instructions/skills/route/SKILL.md`, cold-start tier `mid`, citing the evidence tally (with its date) per the table's existing citation convention.
- [ ] 3.2 Confirm the baseline test from 2.1 now passes.

## 4. REFACTOR + close out

- [ ] 4.1 Tidy any examples or surrounding text in `route/SKILL.md` the new row affects, per `writing-skills`'s pressure-test step.
- [ ] 4.2 Run `targ check-full`.
- [ ] 4.3 Close toejough/engram#722 referencing the commit.
