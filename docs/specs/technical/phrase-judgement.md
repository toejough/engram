# Phrase Judgement - Technical Spec

## 855d3b Orientation

The command `engram auto phrases`. Turns the [depth](../../GLOSSARY.md#depth-recall-moment) of the moment judgement into the request of the mid-tier judgement of the situation phrases; tested through standard input and output. Called by [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow).

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
