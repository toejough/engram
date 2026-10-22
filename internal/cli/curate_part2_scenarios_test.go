package cli_test

// Curate, part 2 (792a22n The outcome of a curation is not reported back; 792a22o Pending offers are surfaced at
// three points). Each scenario drives the surface it names through the command path: the served learn route, the
// query command and the served query route, the update report, and the write commands whose warning is the
// nudge. The vaults are in memory (curate1Machine, curate_part1_scenarios_test.go, and the machine of the update
// scenarios, install_and_update_part2_scenarios_test.go), so no real file system, home or network is touched.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// Test792a22n1a_ServeLearn_TheResponseIsAReceiptAndNothingReportsTheDisposition: a served learn that creates a
// pending offer is answered with a receipt that confirms only that the offer was received (no disposition field),
// while the offer is still pending; then the host curates the offer, and neither the server nor the host sends
// anything to the caller or reports the disposition anywhere the caller reads.
func Test792a22n1a_ServeLearn_TheResponseIsAReceiptAndNothingReportsTheDisposition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newCurate1Machine(t, false)

	var (
		fetches      atomic.Int32
		serveOut     bytes.Buffer
		serveErr     bytes.Buffer
		curationOut  bytes.Buffer
		curationErr  bytes.Buffer
		countingSend = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
			fetches.Add(1)

			return cli.FetchResponse{}, nil
		}
	)

	serveDeps := machine.deps(&serveOut, &serveErr)
	serveDeps.Fetch = countingSend

	body, marshalErr := json.Marshal(offeredFact("served-offer", "object", offerOf(childOrigin, "key-one", "")))
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	resp := routeFor(t, cli.ServeRoutes(serveDeps, curate1VaultPath, "personal", ""), "/learn").
		Serve(t.Context(), cli.ServeRequest{Body: body})
	g.Expect(resp.Status).To(Equal(200))

	var fields map[string]any

	g.Expect(json.Unmarshal(resp.Body, &fields)).To(Succeed())
	g.Expect(slices.Sorted(maps.Keys(fields))).To(Equal(curate2ReceiptKeys),
		"the response is a receipt: it carries no disposition field")
	g.Expect(fields["status"]).To(Equal("offer received"), "it confirms only that the offer was received")
	g.Expect(fields["pending"]).To(BeTrue())

	offers := machine.noteNames()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	g.Expect(cli.ExportNoteHasPendingMarker([]byte(machine.read(offers[0])))).To(BeTrue(),
		"when the response is back no curation has run")
	g.Expect(fetches.Load()).To(BeZero(), "nothing was sent after the response")

	answeredLog := serveErr.String()

	// Later, on the host, the agent curates the offer.
	curationDeps := machine.deps(&curationOut, &curationErr)
	curationDeps.Fetch = countingSend

	hash := mustHash(t, []byte(machine.read(offers[0])))
	_, executeErr := targ.Execute([]string{
		"engram", "amend", "--target", strings.TrimSuffix(offers[0], ".md"), "--discard", "--expect-hash", hash,
		"--vault", curate1VaultPath,
	}, cli.Targets(curationDeps)...)
	g.Expect(executeErr).NotTo(HaveOccurred())

	if executeErr != nil {
		return
	}

	g.Expect(machine.noteNames()).To(BeEmpty(), "the offer was curated")
	g.Expect(machine.exits).To(BeEmpty())
	g.Expect(fetches.Load()).To(BeZero(), "no message about the outcome reached the offering caller")
	g.Expect(serveOut.String()).To(BeEmpty(), "the server wrote nothing to its output")
	g.Expect(serveErr.String()).To(Equal(answeredLog), "the server logged nothing about the outcome")
}

// Test792a22o1a_Query_TheFlagFollowsWhetherAnOfferIsPending: a vault without a pending offer prints a payload with
// no pending_offers key; the same vault with a pending offer prints pending_offers true; once the offer is
// curated away the flag is gone again.
func Test792a22o1a_Query_TheFlagFollowsWhetherAnOfferIsPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := curate2Vault(t)

	_, without := curate2Query(t, machine)
	g.Expect(without).NotTo(HaveKey("pending_offers"), "no pending offer, no flag")

	curate2PlantOffer(machine)

	_, with := curate2Query(t, machine)
	g.Expect(with).To(HaveKeyWithValue("pending_offers", true), "a pending offer, the flag is true")

	_, discardErr := curate2Run(machine, "amend", "--target", strings.TrimSuffix(curate1OfferName, ".md"),
		"--discard", "--expect-hash", mustHash(t, []byte(machine.read(curate1OfferName))))
	g.Expect(discardErr).NotTo(HaveOccurred())

	if discardErr != nil {
		return
	}

	_, after := curate2Query(t, machine)
	g.Expect(after).NotTo(HaveKey("pending_offers"), "the offer is gone, so is the flag")
}

// Test792a22o1b_Query_TheHintIsTheInstructionTheOtherTwoSurfacesCarry: a vault with a pending offer prints
// pending_offers true and a pending_offers_hint that is the same instruction the update notice and the write-path
// nudge carry.
func Test792a22o1b_Query_TheHintIsTheInstructionTheOtherTwoSurfacesCarry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	surfaces := curate2Surfaces(t)

	g.Expect(surfaces.flag).To(BeTrue(), "pending_offers is true")
	g.Expect(surfaces.hint).NotTo(BeEmpty())
	g.Expect(surfaces.notice).To(HaveSuffix(surfaces.hint), "the update notice ends with the hint's text")
	g.Expect(surfaces.nudge).To(HaveSuffix(surfaces.hint), "the write-path nudge ends with the hint's text")
}

// Test792a22o1c_Query_TheFlagAndHintLeadALargePayload: a payload of more than 100 KB from a vault with a pending
// offer has pending_offers true and pending_offers_hint before items, and the first 1500 bytes hold the hint.
func Test792a22o1c_Query_TheFlagAndHintLeadALargePayload(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := curate2Vault(t)

	for index := range curate2LargeNotes {
		name := curate2LargeName(index)
		machine.put(name, curate1FactNote(curate2Situation, "")+strings.Repeat("x", curate2LargeBody))
		machine.putSidecar(name)
	}

	curate2PlantOffer(machine)

	raw, payload := curate2Query(t, machine)
	g.Expect(len(raw)).To(BeNumerically(">", curate2LargeBytes), "the payload is larger than 100 KB")

	itemsAt := strings.Index(raw, "\nitems:")
	flagAt := strings.Index(raw, "\npending_offers: true\n")
	hintAt := strings.Index(raw, "\npending_offers_hint:")

	g.Expect(flagAt).To(BeNumerically(">", 0))
	g.Expect(hintAt).To(BeNumerically(">", flagAt))
	g.Expect(itemsAt).To(BeNumerically(">", hintAt), "pending_offers and the hint appear before items")
	g.Expect(payload).To(HaveKeyWithValue("pending_offers", true))

	hint, isText := payload["pending_offers_hint"].(string)
	g.Expect(isText).To(BeTrue())
	g.Expect(hint).NotTo(BeEmpty())
	g.Expect(itemsAt).To(BeNumerically("<=", curate2PreviewBytes), "the flag and hint end inside the first 1500 bytes")
	g.Expect(raw[:curate2PreviewBytes]).To(ContainSubstring(curate2Command), "the preview shows the hint's command")
}

// Test792a22o1d_Instruction_SaysCurationIsExpectedUpkeepOnEverySurface: the instruction on the query hint, the
// update notice and the write-path nudge says curating is expected vault upkeep and not an extra, to curate after
// finishing the user's request without asking, and gives the engram query command.
func Test792a22o1d_Instruction_SaysCurationIsExpectedUpkeepOnEverySurface(t *testing.T) {
	t.Parallel()

	surfaces := curate2Surfaces(t)

	for name, text := range map[string]string{
		"query hint":       surfaces.hint,
		"update notice":    surfaces.notice,
		"write-path nudge": surfaces.nudge,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(text).To(ContainSubstring("expected vault upkeep, not an extra"))
			g.Expect(text).To(ContainSubstring("after you finish the user's request"))
			g.Expect(text).To(ContainSubstring("without asking"))
			g.Expect(text).To(ContainSubstring(curate2Command))
		})
	}
}

// Test792a22o1e_Query_NoOfferMeansNoFlagAndNoHint: a vault with no pending offer prints a payload with neither
// pending_offers nor pending_offers_hint, byte-identical to the payload the same vault prints with the two keys
// taken away from the pending-offer form.
func Test792a22o1e_Query_NoOfferMeansNoFlagAndNoHint(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := curate2Vault(t)

	clean, cleanPayload := curate2Query(t, machine)
	g.Expect(clean).NotTo(ContainSubstring("pending_offers"))
	g.Expect(cleanPayload).NotTo(HaveKey("pending_offers"))
	g.Expect(cleanPayload).NotTo(HaveKey("pending_offers_hint"))

	curate2PlantOffer(machine)

	flagged, _ := curate2Query(t, machine)
	g.Expect(flagged).To(ContainSubstring("pending_offers_hint:"))

	kept := make([]string, 0, strings.Count(flagged, "\n"))

	for _, line := range strings.SplitAfter(flagged, "\n") {
		if !strings.HasPrefix(line, "pending_offers") {
			kept = append(kept, line)
		}
	}

	g.Expect(clean).To(Equal(strings.Join(kept, "")),
		"the payload without offers is the payload as it was before the hint")
}

// Test792a22o1f_ServeQuery_ARemoteQueryCarriesTheHint: a served query while a pending offer exists on the host is
// answered with the YAML the local query prints, with pending_offers true and the hint.
func Test792a22o1f_ServeQuery_ARemoteQueryCarriesTheHint(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := curate2Vault(t)
	curate2PlantOffer(machine)

	local, localPayload := curate2Query(t, machine)
	g.Expect(localPayload).To(HaveKeyWithValue("pending_offers", true))

	routes := cli.ServeRoutes(machine.deps(&bytes.Buffer{}, &bytes.Buffer{}), curate1VaultPath, "personal",
		curate1ChunksPath)
	resp := routeFor(t, routes, "/query").Serve(t.Context(), cli.ServeRequest{
		Query: map[string][]string{"phrase": {curate2Situation}},
	})

	g.Expect(resp.Status).To(Equal(200), string(resp.Body))
	g.Expect(string(resp.Body)).To(Equal(local), "the response body is the YAML the local query prints")

	var served map[string]any

	g.Expect(yaml.Unmarshal(resp.Body, &served)).To(Succeed())
	g.Expect(served).To(HaveKeyWithValue("pending_offers", true))
	g.Expect(served).To(HaveKey("pending_offers_hint"))
}

// Test792a22o1g_Update_TheReportNamesThePendingOffersAndTheCommand: engram update on a machine whose vault holds a
// pending offer prints a notice line that names the pending offers and the engram query command, and leaves the
// offer where it was (the notice is all update does about it); a vault with no pending offer prints no such line.
func Test792a22o1g_Update_TheReportNamesThePendingOffersAndTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.fileSystem.seedFile(iu3Vault+"/"+curate1OfferName, curate1FactNote(curate2Situation, "pending: true\n"))
	offerBefore := machine.fileSystem.text(iu3Vault + "/" + curate1OfferName)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	notices := curate2LinesWith(outcome.stdout, curate2NoticeLead)
	g.Expect(notices).To(HaveLen(1), "one notice line")

	if len(notices) != 1 {
		return
	}

	g.Expect(notices[0]).To(HavePrefix(curate2NoticeLead), "the notice names the pending offers")
	g.Expect(notices[0]).To(ContainSubstring(curate2Command))
	g.Expect(machine.fileSystem.text(iu3Vault+"/"+curate1OfferName)).To(Equal(offerBefore), "update does not curate")

	quiet := iu3NewMachine()
	quiet.fileSystem.seedFile(iu3Vault+"/"+curate1LiveName, curate1FactNote(curate2Situation, ""))

	quietOutcome := quiet.update(t, cli.UpdateArgs{})
	g.Expect(quietOutcome.err).NotTo(HaveOccurred())

	if quietOutcome.err != nil {
		return
	}

	g.Expect(quietOutcome.stdout).NotTo(ContainSubstring("pending offer"))
}

// Test792a22o1h_WriteCommands_ACompletedWriteWarnsOfPendingOffers: engram learn, engram amend and engram
// resituate on a vault that holds a pending offer complete and log a warning line that notes the pending offers
// and carries the query; on a vault with none they log no such line.
func Test792a22o1h_WriteCommands_ACompletedWriteWarnsOfPendingOffers(t *testing.T) {
	t.Parallel()

	commands := map[string][]string{
		"learn": {
			"learn", "fact", "--slug", "new-fact", "--position", "top", "--source", "test",
			"--situation", "writing a new fact", "--subject", "engram", "--predicate", "warns", "--object", "writers",
		},
		"amend":     {"amend", "--target", strings.TrimSuffix(curate1LiveName, ".md"), "--object", "changed"},
		"resituate": {"resituate", "--note", strings.TrimSuffix(curate1LiveName, ".md"), "--situation", "elsewhere"},
	}

	for name, args := range commands {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			machine := curate2Vault(t)
			curate2PlantOffer(machine)

			stderr, err := curate2Run(machine, args...)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(machine.exits).To(BeEmpty())

			nudges := curate2LinesWith(stderr, curate2NudgeLead)
			g.Expect(nudges).To(HaveLen(1), "one warning line: %s", stderr)

			if len(nudges) != 1 {
				return
			}

			g.Expect(nudges[0]).To(ContainSubstring(curate2Command), "the line carries the same query")

			clean := curate2Vault(t)

			cleanStderr, cleanErr := curate2Run(clean, args...)
			g.Expect(cleanErr).NotTo(HaveOccurred())

			if cleanErr != nil {
				return
			}

			g.Expect(cleanStderr).NotTo(ContainSubstring("pending offer"), "no offer, no warning")
		})
	}
}

// Test792a22o1i_Query_TheNoticesCommandReturnsTheCurateRunbookByTrigger: a vault that holds the curate runbook,
// asked with the command each of the three surfaces prints, returns the runbook first with provenance trigger.
func Test792a22o1i_Query_TheNoticesCommandReturnsTheCurateRunbookByTrigger(t *testing.T) {
	t.Parallel()

	surfaces := curate2Surfaces(t)

	for name, text := range map[string]string{
		"query hint":       surfaces.hint,
		"update notice":    surfaces.notice,
		"write-path nudge": surfaces.nudge,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			command := curate2CommandInText.FindStringSubmatch(text)
			g.Expect(command).To(HaveLen(2), "the surface prints a command")

			if len(command) != 2 {
				return
			}

			textFlag := curate2TextFlag.FindStringSubmatch(command[1])
			phraseFlag := curate2PhraseFlag.FindStringSubmatch(command[1])
			g.Expect(textFlag).To(HaveLen(2), "the command has --text")
			g.Expect(phraseFlag).To(HaveLen(2), "the command has --phrase")

			if len(textFlag) != 2 || len(phraseFlag) != 2 {
				return
			}

			machine := newCurate1Machine(t, false)
			machine.put(curate2RunbookName, curateSkillNoteFixture("abc123"))
			machine.putSidecar(curate2RunbookName)
			machine.put(curate1LiveName, curate1FactNote("curating a garden", ""))
			machine.putSidecar(curate1LiveName)

			stdout, stderr := machine.run("query", "--text="+textFlag[1], "--phrase="+phraseFlag[1],
				"--chunks-dir", curate1ChunksPath)
			g.Expect(stderr).To(BeEmpty())

			var payload queryParsed

			g.Expect(yaml.Unmarshal([]byte(stdout), &payload)).To(Succeed())
			g.Expect(payload.Items).NotTo(BeEmpty())

			if len(payload.Items) == 0 {
				return
			}

			g.Expect(payload.Items[0].Path).To(Equal(curate2RunbookName), "the curate runbook is first")
			g.Expect(payload.Items[0].Kind).To(Equal("runbook"))
			g.Expect(payload.Items[0].Provenances).To(ContainElement("trigger"))
		})
	}
}

// unexported constants.
const (
	// curate2Command is the command every surface carries.
	curate2Command = `engram query --text="curate pending offers"`
	// curate2LargeBody pads a note so that curate2LargeNotes of them pass 100 KB.
	curate2LargeBody    = 6000
	curate2LargeBytes   = 100_000
	curate2LargeFirstID = 10
	curate2LargeNotes   = 25
	// curate2NoticeLead and curate2NudgeLead begin the update notice and the write-path warning.
	curate2NoticeLead   = "vault holds pending offer(s) awaiting curation"
	curate2NudgeLead    = "warning: vault holds pending offer(s) awaiting curation"
	curate2PreviewBytes = 1500
	curate2RunbookName  = "1049.2026-09-21.skill-claude-curate.md"
	curate2Situation    = "releasing a module"
)

// unexported variables.
var (
	curate2CommandInText = regexp.MustCompile("`(engram query --text=[^`]*)`")
	curate2PhraseFlag    = regexp.MustCompile(`--phrase="([^"]*)"`)
	curate2ReceiptKeys   = []string{"basename", "luhmann", "pending", "status", "stored_hash", "vault_id"}
	curate2TextFlag      = regexp.MustCompile(`--text="([^"]*)"`)
)

// curate2Instructions is the curate instruction as each of the three surfaces prints it.
type curate2Instructions struct {
	// flag is the payload's pending_offers value.
	flag bool
	// hint is the payload's pending_offers_hint.
	hint string
	// notice is the update report's notice line, nudge the write-path warning line.
	notice string
	nudge  string
}

// curate2FirstLine is the first line of text that begins with lead, "" when there is none.
func curate2FirstLine(text, lead string) string {
	lines := curate2LinesWith(text, lead)
	if len(lines) == 0 {
		return ""
	}

	return lines[0]
}

// curate2LargeName is the file name of one of the large notes.
func curate2LargeName(index int) string {
	return fmt.Sprintf("%d.2026-01-01.large-%d.md", curate2LargeFirstID+index, index)
}

// curate2LinesWith lists the lines of text that begin with lead.
func curate2LinesWith(text, lead string) []string {
	lines := make([]string, 0, 1)

	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, lead) {
			lines = append(lines, line)
		}
	}

	return lines
}

// curate2PlantOffer puts a pending offer into the machine's vault.
func curate2PlantOffer(machine *curate1Machine) {
	machine.put(curate1OfferName, curate1FactNote("offered "+curate2Situation, "pending: true\n"))
	machine.putSidecar(curate1OfferName)
}

// curate2Query runs `engram query` on the machine's vault and returns the printed payload and its top-level keys.
func curate2Query(t *testing.T, machine *curate1Machine) (string, map[string]any) {
	t.Helper()

	stdout, stderr := machine.run("query", "--phrase", curate2Situation, "--chunks-dir", curate1ChunksPath)
	if stderr != "" || len(machine.exits) != 0 {
		t.Fatalf("query: stderr %q, exits %v", stderr, machine.exits)
	}

	payload := map[string]any{}

	err := yaml.Unmarshal([]byte(stdout), &payload)
	if err != nil {
		t.Fatalf("query payload: %v", err)
	}

	return stdout, payload
}

// curate2Run runs a command on the machine's vault and returns what it wrote to stderr.
func curate2Run(machine *curate1Machine, args ...string) (string, error) {
	machine.t.Helper()

	var stdout, stderr bytes.Buffer

	command := append(append([]string{"engram"}, args...), "--vault", curate1VaultPath)

	_, err := targ.Execute(command, cli.Targets(machine.deps(&stdout, &stderr))...)

	return stderr.String(), err
}

// curate2Surfaces reads the instruction from all three surfaces of a vault that holds a pending offer.
func curate2Surfaces(t *testing.T) curate2Instructions {
	t.Helper()

	machine := curate2Vault(t)
	curate2PlantOffer(machine)

	_, payload := curate2Query(t, machine)
	flag, _ := payload["pending_offers"].(bool)
	hint, _ := payload["pending_offers_hint"].(string)

	stderr, err := curate2Run(machine, "learn", "fact", "--slug", "surface", "--position", "top", "--source", "test",
		"--situation", "reading the surfaces", "--subject", "engram", "--predicate", "prints", "--object", "a nudge")
	if err != nil {
		t.Fatalf("learn: %v", err)
	}

	updating := iu3NewMachine()
	updating.fileSystem.seedFile(iu3Vault+"/"+curate1OfferName, curate1FactNote(curate2Situation, "pending: true\n"))

	outcome := updating.update(t, cli.UpdateArgs{})
	if outcome.err != nil {
		t.Fatalf("update: %v", outcome.err)
	}

	return curate2Instructions{
		flag: flag, hint: hint,
		notice: curate2FirstLine(outcome.stdout, curate2NoticeLead), nudge: curate2FirstLine(stderr, curate2NudgeLead),
	}
}

// curate2Vault is an in-memory vault with one live fact about curate2Situation.
func curate2Vault(t *testing.T) *curate1Machine {
	t.Helper()

	machine := newCurate1Machine(t, false)
	machine.put(curate1LiveName, curate1FactNote(curate2Situation, ""))
	machine.putSidecar(curate1LiveName)

	return machine
}
