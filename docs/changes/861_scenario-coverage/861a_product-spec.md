# Scenario Coverage - Product Spec

## 861a1 Problem statement

The owner and every agent that relies on the specifications cannot tell which documented behavior is verified. Of the 3,562 scenarios in `docs/specs`, 1,655 have no test traced to them (a scenario has a test when a top-level function `Test<ID>_...` exists in some `*_test.go`), and 19 test functions name an ID that no specification holds. A regression in an untraced scenario goes unseen, and a promise of the record cannot be told from an untested one. The owner needs every documented scenario traced to a test that fails when the promise breaks, or to a recorded reason why none exists, without changing what the software promises.

## 861a2 References

This change adds no outcome and no promise. The scenarios in scope belong to capabilities and components that realize the outcomes [858a2a say-once](../858_documentation-framework-adoption/858a_product-spec.md#858a2a-added-say-once) to [858a2h first-pass-success](../858_documentation-framework-adoption/858a_product-spec.md#858a2h-added-first-pass-success). They are listed by owner file in [861c2 Scenario tests by owner file and package](./861c_implementation-plan.md#861c2-scenario-tests-by-owner-file-and-package), with the blocked kinds in [861c3 Blocked nodes](./861c_implementation-plan.md#861c3-blocked-nodes). The gap record this change works from is [858c2a Scenarios without a test, by owner file](../858_documentation-framework-adoption/858c_implementation-plan.md#858c2a-scenarios-without-a-test-by-owner-file) and [858c2b Test functions whose ID is in no spec](../858_documentation-framework-adoption/858c_implementation-plan.md#858c2b-test-functions-whose-id-is-in-no-spec); the issue is [#861](https://github.com/toejough/engram/issues/861).

## 861a3 Operations on the product specs

Empty until a node finds a documentation problem in a product spec; each Modified or Added operation is then recorded here and applied in place to `docs/specs` in the commit that adds its test.
