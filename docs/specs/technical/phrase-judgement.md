# Phrase Judgement - Technical Spec

## 855d3b Phrase Judgement

The command `engram auto phrases`. Turns the [depth](../../GLOSSARY.md#depth-recall-moment) of the moment judgement into the request of the mid-tier judgement of the situation phrases; tested through standard input and output. Called by [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow).

- Ownership: repository-owned component (the command `engram auto phrases`).
- Qualification: Purpose: the [request](../../GLOSSARY.md#request-automatic-memory) that has a model write the lookup phrases that the [recall skill](../../GLOSSARY.md#recall-skill)'s Step 1 asks for, which [793a2b Memory arrives with the prompt](../product/automatic-memory.md#793a2b-memory-arrives-with-the-prompt) needs so that the [notes](../../GLOSSARY.md#permanent-note) and [passages](../../GLOSSARY.md#passage) that arrive are the ones the recall [skill](../../GLOSSARY.md#skill) draws on, and [855a2a Delivery writes what its depth writes](../product/automatic-memory.md#855a2a-delivery-writes-what-its-depth-writes) needs for a deep delivery. Contract: [855d4v Phrasing the lookup](#855d4v-phrasing-the-lookup) is the command's own, apart from the flow that calls it ([793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow)), and its scenarios run the command directly through standard input and output. Identity: the requirement states only what the command reads, prints and fails with, not a function or an injected collaborator, so a refactoring or a change of injection leaves the boundary as it is.
- Required behavior: [Moment Judgement](./moment-judgement.md) is assumed to ask for a [reply](../../GLOSSARY.md#reply-automatic-memory) of the shape `{"depth":...}` and to hand on a [carry](../../GLOSSARY.md#carry) that holds the [cue](../../GLOSSARY.md#cue) and the window ([855d4a Judging the moment](./moment-judgement.md#855d4a-judging-the-moment)); compatibility verification: none beyond the replies and carries written by hand in [855d4v1 Scenarios](#855d4v1-scenarios).
- Required behavior: [Recall](../product/recall.md): the recall skill is assumed to ask, as [792a14e Ten phrases from ten angles](../product/recall.md#792a14e-ten-phrases-from-ten-angles) states, for one short phrase from each of the ten angles that the instructions of the request repeat; compatibility verification: none, the repeated text is `internal/auto/prompts/phrases.txt`, which no test compares with the skill.
- Supported capabilities: [Automatic Memory](../product/automatic-memory.md).

## 855d4v Phrasing the lookup

Requirement: 855d4v The `phrase-judgement` MUST read one JSON object from standard input: the `session`'s ID, the `reply` of the moment judgement and its `carry`, which holds the cue. It MUST [read the reply](../../GLOSSARY.md#reply-automatic-memory) as a JSON object with a `depth` of `none`, `glance` or `deep`, and fail, printing nothing and naming the judgement, when the reply is not one or the depth is none of the three. A depth of `none` MUST print nothing, succeed and look nothing up. Otherwise it MUST print one object `{"request":{...},"carry":{...}}`. The request has `model` of the mid tier, `system`, `prompt` and `maxTokens`, and no `timeoutMs` ([793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow)). The `system` MUST hold the instructions and nothing of the [session](../../GLOSSARY.md#session): the [recall skill](../../GLOSSARY.md#recall-skill)'s Step 1, one short queryable phrase for each of ten angles (the situation or setting, the user's intent, the current concrete action, the problem or blocker, the candidate approach, the tooling in play, prior related work, an adjacent technique, a failure mode to avoid, the domain), about 3 of them at glance and all 10 at deep, each keyed to the task and not to a fear ("implementing Claude Code hooks", not "common mistakes when writing hooks"), none filtered before the lookup, and the reply shape `{"phrases":[...]}`. The `prompt` MUST hold the window of the carry between `<window>` and `</window>`. The carry holds the cue, the depth and the window. It MUST fail, printing nothing and naming the cause, when the input is not a JSON object.

Owner: [Phrase Judgement](./phrase-judgement.md).

Supports: [855a3b1b A note whose words match but that does not apply is not delivered](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1d The notes that arrive are the notes the recall skill draws on](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1e A passage of an earlier conversation arrives when it applies](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a2a1b A deep delivery writes the lesson the vault lacks](../product/automatic-memory.md#855a2a1-uat-scenarios).

### 855d4v1 Scenarios

```gherkin
Scenario: 855d4v1a The request holds the ten angles and the window
  Given a reply of depth deep and a carry that holds a cue and a window, one of them with a tag in it
  When the phrases are asked
  Then it prints a request for the model sonnet with no time limit, whose system names the ten angles, the keying to the task and the reply shape, whose prompt holds the window, and a carry that holds the cue of the carry, the depth and the window, the prompt holding the window escaped again

Scenario: 855d4v1b The number of phrases follows the depth
  Given replies of depth glance and of depth deep
  When the phrases are asked for each
  Then the system of the first asks for about 3 phrases and that of the second for 10

Scenario: 855d4v1c A depth of none looks nothing up
  Given a reply of depth none
  When the phrases are asked
  Then it prints nothing, succeeds and looks nothing up

Scenario: 855d4v1d A fenced reply is read and a reply that is not a moment reply fails the command
  Given a reply of depth glance in a code fence, replies that are not JSON, and one that holds a depth that is none of the three
  When the phrases are asked for each
  Then the first prints a request, and the others print nothing and fail with an error that names the judgement
```
