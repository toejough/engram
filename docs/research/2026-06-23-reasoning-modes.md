# Reasoning modes — definitions + literature examples (test grounding)

> Purpose: a *vetted* example bank, drawn from the logic/philosophy literature, to replace the
> self-authored fixtures that proved guessable. The three classical modes are Peirce's
> (C.S. Peirce, "Deduction, Induction, and Hypothesis", *Popular Science Monthly*, 1878), which he
> illustrated with one bag-of-beans example — reused below so all three are directly comparable.
> Each example lists **premises → conclusion** and the source. Tests (`dev/eval/traps/reasoning_eval.py`)
> use *idiosyncratic instantiations* of each logical form (novel tokens) so we test the reasoning, not
> recall of the famous example.

## Deduction — rule + case ⟹ *necessary* result

The conclusion is guaranteed if the premises are true (truth-preserving; analytic).

1. **Aristotle's syllogism** (Aristotle, *Prior Analytics*): All men are mortal; Socrates is a man; ∴ Socrates is mortal.
2. **Peirce's beans (deductive form)** (Peirce 1878): All the beans in this bag are white; these beans are from this bag; ∴ these beans are white.
3. **Modus ponens** (classical logic): If it rains, the ground is wet; it is raining; ∴ the ground is wet.
4. **Geometry** (Euclid, textbook): All squares have four equal sides; this figure is a square; ∴ this figure has four equal sides.
5. **Transitivity of order** (classical): If A is taller than B and B is taller than C, then A is taller than C; A is taller than B; B is taller than C; ∴ A is taller than C.

## Induction — cases + results ⟹ *probable* general rule

Generalizes from observed instances; ampliative, not truth-preserving (the conclusion can fail).

1. **Peirce's beans (inductive form)** (Peirce 1878): These beans are from this bag; these beans are white; ∴ (probably) all the beans in this bag are white.
2. **Hume's sunrise** (Hume, *Enquiry Concerning Human Understanding*): The sun has risen every day in recorded history; ∴ the sun will rise tomorrow.
3. **Black crows** (classic textbook): Every crow observed so far has been black; ∴ all crows are black.
4. **Swans / falsifiability** (Popper, *Logik der Forschung*): Every swan observed in Europe was white; ∴ all swans are white (later falsified by black swans in Australia — induction's defeasibility).
5. **Thermal expansion** (enumerative, science textbook): This iron bar expanded when heated; this copper bar expanded when heated; this silver bar expanded when heated; ∴ metals expand when heated.

## Abduction — result + rule ⟹ *best explanation* (hypothesis)

Infers the most plausible cause/explanation of an observation; ampliative, defeasible ("inference to the best explanation").

1. **Peirce's beans (abductive form)** (Peirce 1878 — the canonical abduction example): All the beans in this bag are white; these (loose) beans are white; ∴ these beans are (probably) from this bag.
2. **Wet grass** (classic IBE): The grass is wet this morning; rain would make the grass wet; ∴ it (probably) rained overnight.
3. **Holmes on Watson** (Doyle, *A Study in Scarlet* — popularised abduction): a newcomer has a doctor's air, a military bearing, a tanned face, and a stiff arm; recent army-doctor service in the tropics would explain all four; ∴ "you have been in Afghanistan."
4. **Diagnosis** (medical IBE, textbook): the patient has fever, productive cough, and a lung opacity on X-ray; pneumonia would produce exactly these; ∴ the patient probably has pneumonia.
5. **Footprints** (classic): there are fresh footprints across the snow; someone walking here would leave them; ∴ someone walked here recently.

## Result (2026-06-23, opus, n=3, idiosyncratic instantiations, no memory)

| mode | hit (conclusion + correct certainty) |
|---|---|
| deduction (necessary) | 15/15 |
| induction (probable) | 15/15 |
| abduction (best explanation) | 14/15 |

Opus reasons soundly across all three canonical modes on novel content, with correct certainty
calibration (asserts deduction as necessary; hedges induction/abduction as probable). **Notable:** on
the weak Peirce beans-abduction it initially "failed" because it *correctly refused* the inference —
"opaque isn't exclusive to jar-K, so concluding 'from jar-K' affirms the consequent." That was the
answer key being wrong, not opus; the case was rewritten to test fallacy-detection (now 3/3). The lone
remaining miss is one noisy Holmes run. **Implication for engram:** the reasoning is not the gap — opus
does deduction/induction/abduction well given the facts; the lever is delivering the right facts
(memory). This is the clean, contamination-free instrument the messier synthesis/compounding evals
lacked; build future memory tests on these vetted forms.

## How the tests use these

For each mode we instantiate the *logical form* with invented tokens (e.g. "all glorbs in crate-Q are fuzzy") so the answer can't be recalled from the famous example, then check: (a) the agent reaches the **correct conclusion** for that form, and (b) it does **not over/under-claim** the mode's certainty (deduction = necessary; induction/abduction = probable, defeasible). A deductive item has exactly one correct answer; an inductive/abductive item's conclusion must be hedged as probable, and a *distractor* alternative must be correctly rejected or ranked lower.
