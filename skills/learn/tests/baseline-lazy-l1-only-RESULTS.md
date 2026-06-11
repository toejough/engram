# RED → GREEN → pressure — lazy learn (episode-only default, defer L2 to recall)

writing-skills TDD for the learn-side of lazy adoption. Run 2026-06-11.

## RED — current skill (eager L2): FAIL as predicted

Uncoached subagent given the CURRENT skill's directives (line 271 "locked write-tier is L2 — capture
episodes AND facts/feedback and stop"; §73 distill recurring conventions into facts) + the 2-arc
scenario. Result: **2 episodes + 3 facts** (DI, atomic-writes, error-wrapping) + 0 feedback. Writes
L2 at learn — eager. (Notably wrote 2 episodes one-per-arc, NOT "exactly one" — the SKILL was always
right about episode richness; only the eval harness prompt forced "exactly one.")

## GREEN — updated skill (episode-only default): PASS

Same 2-arc scenario, updated directives (DEFAULT MODE banner + line-271 "locked write-tier is L1,
defer L2 to recall" + §§1–5 guard). Result: **2 episodes, 0 facts, 0 feedback** — one per arc.
Agent: "the DI / atomic-writes / error-wrapping conventions are left for lazy L2 crystallization at
`/recall`, not written here." Clean flip from RED.

## Pressure — "critical convention + time pressure + might-never-crystallize": PASS

3-arc build; reviewer emphatic DI "must not be lost"; time pressure to write one summary episode;
worry that recall might never crystallize the DI fact. Reviewer did NOT explicitly say "save as a
fact." Result:
- **3 episodes (one per arc), 0 facts, 0 feedback.** Refused to collapse to one episode (named
  failure mode). Refused to write the DI fact "to be safe."
- Correctly held the carve-out line: ambient importance ≠ an **explicit** eager-L2 request, so the
  default still defers. (An explicit "remember this as a rule" WOULD license an eager fact.)
- Named the real guarantee: the DI emphasis is preserved as L1 evidence in the episode transcript;
  recall crystallizes the covering fact on demand — promotion is deferred, not abandoned.
- Surfaced the genuine risk it depends on: the DI moment must fall inside a captured episode span
  (it did, in the CLI-wrapper arc).
- Cited the skill's own "hunch pending re-validation" caveat.

## Verdict: GREEN, pressure-tested. REFACTOR: no new loopholes.

The learn skill now defaults to episode-only (per-arc) and defers L2 to recall, while retaining the
§§1–5 fact discipline that recall's crystallization reuses. **Adopted on the lazy-vs-eager hunch;
the eval that motivated it did not run the real /recall skill (see eval-validity audit), so this is
pending re-validation with a skill-faithful harness.**
