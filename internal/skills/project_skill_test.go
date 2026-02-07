package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestProjectSkill_TwoRoleArchitecture verifies TASK-1 AC-7: SKILL.md documents two-role split
// Traces to: ARCH-042, REQ-016
func TestProjectSkill_TwoRoleArchitecture(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document two-role architecture
	g.Expect(text).To(ContainSubstring("Two-Role"), "should document two-role architecture")
	g.Expect(text).To(ContainSubstring("Team Lead"), "should document team lead role")
	g.Expect(text).To(ContainSubstring("Orchestrator"), "should document orchestrator role")
}

// TestProjectSkill_TeamLeadDelegationMode verifies TASK-1 AC-7: SKILL.md documents delegation-only mode
// Traces to: ARCH-050, REQ-016
func TestProjectSkill_TeamLeadDelegationMode(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document delegation-only mode
	g.Expect(text).To(ContainSubstring("delegate"), "should mention delegation mode")
	g.Expect(text).To(ContainSubstring("never edits files"), "should document team lead never edits files directly")

	// Should document prohibited actions
	g.Expect(text).To(MatchRegexp("(?i)(do not|prohibited).*(write|edit)"), "should prohibit direct file editing")
}

// TestProjectSkill_OrchestratorSpawnSequence verifies TASK-1 AC-1: Team lead spawns haiku orchestrator
// Traces to: ARCH-048, REQ-016, REQ-017
func TestProjectSkill_OrchestratorSpawnSequence(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document orchestrator spawn
	g.Expect(text).To(ContainSubstring("spawn orchestrator"), "should document spawning orchestrator teammate")
	g.Expect(text).To(ContainSubstring("haiku"), "should specify haiku model for orchestrator")

	// Should document spawn sequence
	g.Expect(text).To(ContainSubstring("TeamCreate"), "should document TeamCreate call")
	g.Expect(text).To(MatchRegexp("(?i)task.*orchestrator"), "should document spawning orchestrator via Task tool")
}

// TestProjectSkill_SpawnRequestProtocol verifies TASK-1 AC-3: Orchestrator sends spawn requests via SendMessage
// Traces to: ARCH-043, REQ-017, REQ-021
func TestProjectSkill_SpawnRequestProtocol(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document spawn request protocol
	g.Expect(text).To(ContainSubstring("spawn request"), "should document spawn request protocol")
	g.Expect(text).To(ContainSubstring("SendMessage"), "should document using SendMessage for spawn requests")
	g.Expect(text).To(ContainSubstring("task_params"), "should document task_params in spawn requests")
}

// TestProjectSkill_ModelHandshakeValidation verifies TASK-1 AC-1: Team lead validates model handshake
// Traces to: ARCH-047, ARCH-048, REQ-017
func TestProjectSkill_ModelHandshakeValidation(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document model handshake
	g.Expect(text).To(ContainSubstring("handshake"), "should document model handshake validation")
	g.Expect(text).To(MatchRegexp("(?i)validate.*model|model.*validat"), "should document validating model after spawn")
}

// TestProjectSkill_ShutdownProtocol verifies TASK-1 AC-5,6: Shutdown request and end-of-command sequence
// Traces to: ARCH-044, REQ-018
func TestProjectSkill_ShutdownProtocol(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document shutdown protocol
	g.Expect(text).To(ContainSubstring("shutdown"), "should document shutdown protocol")
	g.Expect(text).To(ContainSubstring("all-complete"), "should document all-complete message")
	g.Expect(text).To(ContainSubstring("TeamDelete"), "should document TeamDelete call")

	// Should document end-of-command sequence
	g.Expect(text).To(MatchRegexp("(?i)end.of.command|completion sequence"), "should document end-of-command sequence")
}

// TestProjectSkill_StatePersistenceOwnership verifies TASK-1 AC-2: Orchestrator owns state persistence
// Traces to: ARCH-045, REQ-020, REQ-022
func TestProjectSkill_StatePersistenceOwnership(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document orchestrator owns state
	g.Expect(text).To(MatchRegexp("(?i)orchestrator.*(owns|manages).*state|state.*(owned|managed).*orchestrator"), "should document orchestrator owns state persistence")
	g.Expect(text).To(ContainSubstring("projctl state"), "should reference projctl state commands")
}

// TestProjectSkill_OrchestratorStepLoop verifies TASK-1 AC-2: Orchestrator runs step loop
// Traces to: ARCH-042, REQ-016
func TestProjectSkill_OrchestratorStepLoop(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document orchestrator runs step loop
	g.Expect(text).To(ContainSubstring("step loop"), "should document orchestrator runs step loop")
	g.Expect(text).To(ContainSubstring("projctl step next"), "should reference projctl step next")
	g.Expect(text).To(ContainSubstring("projctl step complete"), "should reference projctl step complete")
}

// TestProjectSkillFull_DetailedArchitecture verifies SKILL-full.md has detailed two-role documentation
// Traces to: ARCH-051, REQ-016
func TestProjectSkillFull_DetailedArchitecture(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should have detailed orchestrator behavior section
	g.Expect(text).To(ContainSubstring("orchestrator"), "should document orchestrator behavior")

	// Should document state persistence ownership
	g.Expect(text).To(MatchRegexp("(?i)state.*persistence|persist.*state"), "should document state persistence")

	// Should document resumption flow
	g.Expect(text).To(MatchRegexp("(?i)resumption|resume"), "should document resumption flow")

	// Should document error handling
	g.Expect(text).To(MatchRegexp("(?i)error.*handling|retry|backoff"), "should document error handling and retry-backoff")
}

// TestProjectSkill_NoOldOrchestratorPattern verifies old single-role pattern is removed
// Traces to: ARCH-042, REQ-016
func TestProjectSkill_NoOldOrchestratorPattern(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// The old pattern had team lead running the step loop directly
	// New pattern has team lead spawn orchestrator who runs the loop
	// We shouldn't see both patterns mixed together - either it's documented
	// that team lead delegates to orchestrator, or team lead is running loop

	// Check for the key indicator: if "step loop" is mentioned,
	// it should be in the context of orchestrator, not team lead
	if strings.Contains(text, "step loop") {
		// Find the section containing "step loop"
		// It should mention orchestrator/teammate, not "you run" or "team lead runs"
		g.Expect(text).ToNot(MatchRegexp("(?i)(you run|team lead runs).*step loop"), "team lead should not run step loop directly")
	}
}

// TestProjectSkill_TeamLeadSpawnConfirmation verifies TASK-1 AC-4: Team lead confirms spawns
// Traces to: ARCH-043, REQ-017
func TestProjectSkill_TeamLeadSpawnConfirmation(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document spawn confirmation
	g.Expect(text).To(MatchRegexp("(?i)confirm.*spawn|spawn.*confirm"), "should document confirming spawns to orchestrator")
}

// TestProjectSkill_TeamLeadCallsTeamCreate verifies TASK-2 AC-1: Team lead calls TeamCreate with project name
// Traces to: ARCH-048, ARCH-042, REQ-016
func TestProjectSkill_TeamLeadCallsTeamCreate(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document calling TeamCreate on /project invocation
	g.Expect(text).To(ContainSubstring("TeamCreate"), "should document TeamCreate call")
	g.Expect(text).To(MatchRegexp("(?i)TeamCreate.*project.*name|project.*name.*TeamCreate"), "should document passing project name to TeamCreate")
	g.Expect(text).To(MatchRegexp("(?i)/project.*invocation|invoke.*project"), "should document /project invocation trigger")
}

// TestProjectSkill_TeamLeadSpawnsOrchestratorHaiku verifies TASK-2 AC-2: Team lead spawns orchestrator with model=haiku
// Traces to: ARCH-048, REQ-016, REQ-017
func TestProjectSkill_TeamLeadSpawnsOrchestratorHaiku(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document using Task tool to spawn orchestrator
	g.Expect(text).To(MatchRegexp("(?i)Task.*tool.*orchestrator|spawn.*orchestrator.*Task"), "should document spawning orchestrator via Task tool")

	// Should document model=haiku for orchestrator
	g.Expect(text).To(MatchRegexp("(?i)model.*haiku|haiku.*model"), "should document haiku model for orchestrator")

	// Should document orchestrator teammate name
	g.Expect(text).To(MatchRegexp("(?i)name.*orchestrator|orchestrator.*name"), "should document orchestrator as teammate name")
}

// TestProjectSkill_SpawnPromptContents verifies TASK-2 AC-3: Spawn prompt includes required context
// Traces to: ARCH-048, REQ-017
func TestProjectSkill_SpawnPromptContents(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document passing project name in spawn (via TeamCreate team_name or state init --name)
	g.Expect(text).To(MatchRegexp("(?i)(team_name.*project|project.*name|--name)"), "should document passing project name in spawn context")

	// Should document passing issue number/context (via state init --issue)
	g.Expect(text).To(MatchRegexp("(?i)(--issue|issue.*ISSUE-)"), "should document passing issue context in spawn")

	// Should document orchestrator runs step loop
	g.Expect(text).To(MatchRegexp("(?i)(step loop|step.*driven.*loop)"), "should document orchestrator runs step loop")
}

// TestProjectSkill_TeamLeadIdleAfterSpawn verifies TASK-2 AC-4: Team lead enters idle state after spawn
// Traces to: ARCH-048, REQ-016
func TestProjectSkill_TeamLeadIdleAfterSpawn(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document team lead enters idle state
	g.Expect(text).To(MatchRegexp("(?i)idle.*state|wait.*message|waiting.*orchestrator"), "should document team lead enters idle/waiting state after spawn")
}

// TestProjectSkill_OrchestratorInitAndLoop verifies TASK-2 AC-5: Orchestrator starts with state init and enters step loop
// Traces to: ARCH-048, ARCH-045, REQ-016, REQ-020
func TestProjectSkill_OrchestratorInitAndLoop(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document orchestrator starts with projctl state init
	g.Expect(text).To(ContainSubstring("projctl state init"), "should document orchestrator runs projctl state init on startup")

	// Should document orchestrator enters step loop after init
	g.Expect(text).To(MatchRegexp("(?i)init.*step.*loop|state.*init.*loop"), "should document orchestrator enters step loop after state init")
}

// TestProjectSkill_OrchestratorDetectsSpawnProducer verifies TASK-3 AC-1: Orchestrator detects spawn-producer action
// Traces to: ARCH-043, REQ-017, DES-003
func TestProjectSkill_OrchestratorDetectsSpawnProducer(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document orchestrator detects spawn-producer action from projctl step next
	g.Expect(text).To(MatchRegexp("(?i)spawn-producer.*action|detect.*spawn-producer"), "should document orchestrator detects spawn-producer action")
	g.Expect(text).To(MatchRegexp("(?i)projctl step next.*spawn-producer|spawn-producer.*projctl step next"), "should document spawn-producer comes from projctl step next")
}

// TestProjectSkill_OrchestratorDetectsSpawnQA verifies TASK-3 AC-2: Orchestrator detects spawn-qa action
// Traces to: ARCH-043, REQ-017, DES-003
func TestProjectSkill_OrchestratorDetectsSpawnQA(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document orchestrator detects spawn-qa action from projctl step next
	g.Expect(text).To(MatchRegexp("(?i)spawn-qa.*action|detect.*spawn-qa"), "should document orchestrator detects spawn-qa action")
	g.Expect(text).To(MatchRegexp("(?i)projctl step next.*spawn-qa|spawn-qa.*projctl step next"), "should document spawn-qa comes from projctl step next")
}

// TestProjectSkill_OrchestratorComposesSpawnRequest verifies TASK-3 AC-3: Orchestrator composes SendMessage with spawn_request and task_params
// Traces to: ARCH-043, REQ-017, DES-004
func TestProjectSkill_OrchestratorComposesSpawnRequest(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document orchestrator composes SendMessage with spawn_request field
	g.Expect(text).To(MatchRegexp("(?i)SendMessage.*spawn.*request|compose.*SendMessage.*spawn"), "should document orchestrator uses SendMessage for spawn requests")
	g.Expect(text).To(ContainSubstring("task_params"), "should document spawn request includes task_params JSON")
	g.Expect(text).To(MatchRegexp("(?i)full.*task_params|task_params.*JSON"), "should document including full task_params JSON payload")
}

// TestProjectSkill_SpawnRequestMessageFields verifies TASK-3 AC-4: Message includes expected_model, action, phase fields
// Traces to: ARCH-043, REQ-017, DES-004
func TestProjectSkill_SpawnRequestMessageFields(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document message fields
	g.Expect(text).To(ContainSubstring("expected_model"), "should document expected_model field in spawn request")
	g.Expect(text).To(MatchRegexp("(?i)(action|phase).*field|field.*(action|phase)"), "should document action or phase fields in spawn request")
}

// TestProjectSkill_TeamLeadExtractsTaskParams verifies TASK-3 AC-5: Team lead receives and extracts task_params
// Traces to: ARCH-043, REQ-017, DES-003
func TestProjectSkill_TeamLeadExtractsTaskParams(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document team lead receives spawn request message
	g.Expect(text).To(MatchRegexp("(?i)team lead.*receive.*spawn.*request|receive.*spawn.*request.*message"), "should document team lead receives spawn request message")

	// Should document extracting task_params from message
	g.Expect(text).To(MatchRegexp("(?i)extract.*task_params|task_params.*extract"), "should document extracting task_params from spawn request")
}

// TestProjectSkill_TeamLeadCallsTaskTool verifies TASK-3 AC-6: Team lead calls Task tool with extracted parameters
// Traces to: ARCH-043, REQ-017, DES-003
func TestProjectSkill_TeamLeadCallsTaskTool(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document team lead calls Task tool
	g.Expect(text).To(MatchRegexp("(?i)team lead.*Task.*tool|Task.*tool.*spawn"), "should document team lead calls Task tool")

	// Should document passing extracted task_params to Task tool
	// Looking for patterns like: "Task(subagent_type: ..., name: ..., model: ..., prompt: ...)"
	g.Expect(text).To(MatchRegexp("(?i)(subagent_type|name.*model.*prompt|task_params.*Task)"), "should document passing task_params fields to Task tool")

	// Should specifically mention team_name parameter
	g.Expect(text).To(ContainSubstring("team_name"), "should document team_name parameter in Task tool call")
}

// TestProjectSkill_TeamLeadSendsConfirmation verifies TASK-3 AC-7: Team lead sends confirmation after successful spawn
// Traces to: ARCH-043, REQ-017, DES-005
func TestProjectSkill_TeamLeadSendsConfirmation(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document team lead sends confirmation message
	g.Expect(text).To(MatchRegexp("(?i)(send|sends).*confirmation.*spawn|spawn.*confirmation.*message"), "should document team lead sends spawn confirmation")

	// Should document confirmation sent to orchestrator
	g.Expect(text).To(MatchRegexp("(?i)confirmation.*(to|back to).*orchestrator|orchestrator.*confirmation"), "should document confirmation sent back to orchestrator")
}

// --- ISSUE-104 TASK-6: Error Handling with Retry-Backoff ---

// TestProjectSkillFull_ErrorHandlingRetryLogic verifies TASK-6 AC-1,2: Orchestrator wraps step commands with retry
// Traces to: ARCH-046, REQ-019, REQ-023
func TestProjectSkillFull_ErrorHandlingRetryLogic(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document retry logic for step next
	g.Expect(text).To(MatchRegexp("(?i)(wrap|wraps).*(step next|projctl step next).*retry|retry.*(wrap|wraps).*(step next|projctl step next)"), "should document wrapping projctl step next with retry")

	// Should document retry logic for step complete
	g.Expect(text).To(MatchRegexp("(?i)(wrap|wraps).*(step complete|projctl step complete).*retry|retry.*(wrap|wraps).*(step complete|projctl step complete)"), "should document wrapping projctl step complete with retry")
}

// TestProjectSkillFull_SpawnConfirmationRetry verifies TASK-6 AC-3: Spawn confirmation waits have timeout retry
// Traces to: ARCH-046, REQ-019
func TestProjectSkillFull_SpawnConfirmationRetry(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document timeout/retry for spawn confirmation waits
	g.Expect(text).To(MatchRegexp("(?i)(timeout|wait).*spawn.*confirmation.*(retry|backoff)|spawn.*confirmation.*(timeout|wait).*(retry|backoff)"), "should document timeout/retry for spawn confirmation waits")
}

// TestProjectSkillFull_BackoffDelayPattern verifies TASK-6 AC-4: Backoff delays follow 1s, 2s, 4s pattern
// Traces to: ARCH-046, REQ-019, DES-006
func TestProjectSkillFull_BackoffDelayPattern(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document exponential backoff pattern
	g.Expect(text).To(MatchRegexp("(?i)(backoff|exponential.*backoff)"), "should document backoff/exponential backoff")

	// Should document specific delays: 1s, 2s, 4s
	g.Expect(text).To(MatchRegexp("(?i)1s.*2s.*4s|delays.*1.*2.*4"), "should document backoff delays: 1s, 2s, 4s")
}

// TestProjectSkillFull_ErrorEscalationAfterRetries verifies TASK-6 AC-5: Escalate after 3 failed attempts
// Traces to: ARCH-046, REQ-019, REQ-023
func TestProjectSkillFull_ErrorEscalationAfterRetries(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document escalation after 3 attempts
	g.Expect(text).To(MatchRegexp("(?i)(after|max).*3.*(attempt|tr|fail)"), "should document escalation after 3 failed attempts")

	// Should document sending error message to team lead
	g.Expect(text).To(MatchRegexp("(?i)(send|escalate).*error.*(team lead|team-lead)|orchestrator.*send.*message.*error"), "should document orchestrator sends error message to team lead")
}

// TestProjectSkillFull_ErrorMessageFormat verifies TASK-6 AC-6: Error message includes required fields
// Traces to: ARCH-046, REQ-019, DES-006
func TestProjectSkillFull_ErrorMessageFormat(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document error message includes action
	g.Expect(text).To(MatchRegexp("(?i)error.*(message|field).*(action.*phase|phase.*action)|(includes|contains).*(action.*phase|phase.*action)"), "should document error message includes action and phase")

	// Should document error message includes error output
	g.Expect(text).To(MatchRegexp("(?i)error.*(message|field).*(error.*output|output.*error)|(includes|contains).*(error.*output|output.*error)"), "should document error message includes error output")

	// Should document error message includes retry history
	g.Expect(text).To(MatchRegexp("(?i)error.*(message|field).*(retry.*history|history.*retry|attempt.*history)|(includes|contains).*(retry.*history|history.*retry|attempt.*history)"), "should document error message includes retry history")
}

// TestProjectSkillFull_TeamLeadEscalationToUser verifies TASK-6 AC-7: Team lead escalates to user
// Traces to: ARCH-046, REQ-019, REQ-023
func TestProjectSkillFull_TeamLeadEscalationToUser(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	// Check both SKILL.md and SKILL-full.md
	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	skillFullPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	contentFull, err := os.ReadFile(skillFullPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content) + string(contentFull)

	// Should document team lead escalates errors to user
	g.Expect(text).To(MatchRegexp("(?i)(team lead|team-lead).*(escalate|present).*user|escalate.*error.*user"), "should document team lead escalates errors to user")

	// Should mention AskUserQuestion for escalation
	g.Expect(text).To(MatchRegexp("(?i)AskUserQuestion.*error|error.*AskUserQuestion|escalate.*AskUserQuestion"), "should document using AskUserQuestion for error escalation")
}

// TestProjectSkillFull_RetryLogging verifies TASK-6 AC-8: Orchestrator logs retry attempts
// Traces to: ARCH-046, REQ-019
func TestProjectSkillFull_RetryLogging(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document logging retry attempts
	g.Expect(text).To(MatchRegexp("(?i)(log|logs|logging).*(retry.*attempt|attempt.*retry)|orchestrator.*log.*retry"), "should document orchestrator logs retry attempts")
}

// --- ISSUE-104 TASK-7: State Persistence Ownership - Atomic Writes ---

// TestProjectSkillFull_AtomicStateWrites verifies TASK-7 AC-7: Atomic state writes (temp file + rename)
// Traces to: ARCH-045, REQ-020, REQ-022
func TestProjectSkillFull_AtomicStateWrites(t *testing.T) {
	g := NewWithT(t)

	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(homeDir, ".claude", "skills", "project", "SKILL-full.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	// Should document atomic state writes
	g.Expect(text).To(MatchRegexp("(?i)atomic.*(write|state)|state.*atomic"), "should document atomic state writes")

	// Should document temp file + rename mechanism
	g.Expect(text).To(MatchRegexp("(?i)temp.*file.*rename|temporary.*file.*rename"), "should document temp file + rename mechanism")
}
