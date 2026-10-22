# Test Suite Classification - Product Spec

## 862a1 Problem statement

The owner and every agent that relies on the project's checks cannot rely on the suite classification that the Project Profile states. A test tagged `integration` outside `dev/` runs in no check, because the shared integration leg selects tests by name and no test has the name; an acceptance run without a login fails instead of skipping, which reads as a product failure; and the Profile lists known deviations that no decision row owns. The owner needs every test to run in the check its classification names, a run without a login to skip clearly, and each remaining deviation to be a recorded decision.

## 862a2 References

The issue is [#862](https://github.com/toejough/engram/issues/862) (part of #858); it was recorded at the adoption as [858d99 Test-suite classification deviations](../858_documentation-framework-adoption/858d_questions-and-decisions.md#questions-and-decisions). The rules are the [Test-Suite Classification](../../CONVENTIONS-GUIDE.md#test-suite-classification) section of the Conventions Guide and the test-suite table of the [Project Profile](../../../AGENTS.md#project-profile). The eighteen default-suite tests that use a real temporary directory stay tracked by [#860](https://github.com/toejough/engram/issues/860).

## 862a3 Operations on the product specs

Empty until a node finds a documentation problem in a product spec; each Modified or Added operation is then recorded here and applied in place to `docs/specs` in the commit that adds its test.
