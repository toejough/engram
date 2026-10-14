package auto_test

import (
	"encoding/json"
	stdmaps "maps"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

func Test793d26b1_ABlockOfWrappedLinesIsHiddenOnlyWhenItHoldsDigitsAndLetters(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		block, needle string
		hidden        bool
	}{
		"digits and letters": {"abc-def-ghi-jkl-mno-1pq\nabc-def-ghi-jkl-mno-2pq", "mno-1pq", true},
		"letters only":       {"abc-def-ghi-jkl-mno-xyz\nabc-def-ghi-jkl-mno-uvw", "mno-xyz", false},
	}

	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()

			world.Do(t, captureOf([]any{said("user", "No, use targ.\n"+one.block+"\nthanks")}))

			g.Expect(strings.Contains(world.Requests[0].Prompt, one.needle)).To(Equal(!one.hidden))
		})
	}
}

func Test793d26b1_AGitCommitHashIsNotMistakenForASecret(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	sha := "3f786850e387550fdab836ed7e6dc881de23001b"

	world.Do(t, captureOf([]any{said("user", "No, revert commit "+sha+", use targ.")}))

	g.Expect(world.Requests[0].Prompt).To(ContainSubstring(sha))
}

func Test793d26b1_ASecretInALessonThatTheJudgeWroteNeverReachesTheCommandLine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Model = repliesInOrder(lessonsText(lessonWith(map[string]any{
		"situation": "using key sk-ant-api03-JUDGEWROTEASECRET0123", "action": "export TOKEN=hunter2hunter2",
	})))

	world.Do(t, captureOf(turnMessages()))

	g.Expect(world.Find("learn")).To(HaveLen(1))
	g.Expect(world.Find("learn")[0].Line()).NotTo(ContainSubstring("JUDGEWROTEASECRET"))
	g.Expect(world.Find("learn")[0].Line()).NotTo(ContainSubstring("hunter2"))
}

func Test793d26b1_ASecretInTheWindowNeverReachesTheJudgement(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()

	world.Do(t, captureOf([]any{
		said("user", "No: use the key sk-ant-abcdefghijklmnopqrstuvwx and password=hunter2hunter2"),
	}))

	g.Expect(world.Requests[0].Prompt).NotTo(ContainSubstring("abcdefghijklmnopqrstuvwx"))
	g.Expect(world.Requests[0].Prompt).NotTo(ContainSubstring("hunter2"))
	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("[redacted]"))
}

func Test793d26b1_ASecretOfAKnownShapeNeverReachesTheModelTheCommandLineOrANote(t *testing.T) {
	t.Parallel()

	awsKeyID := unmask("{{AKIA}}") + strings.Repeat("A", awsKeyTail)
	shapes := map[string][2]string{
		"a private key": {
			unmask("{{-----BEGIN}} RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0123456789abcdef\n-----END RSA PRIVATE KEY-----"),
			"MIIEowIBAAKCAQEA0123456789abcdef",
		},
		"a JWT": {
			unmask("{{eyJ}}hbGciOiJIUzI1NiJ9.{{eyJ}}zdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r"),
			"dBjftJeZ4CVPmB92K27uhbUJU1p1r",
		},
		"a provider key":   {"sk-ant-api03-abcdefghijklmnopqrstuvwxyz", "abcdefghijklmnopqrstuvwxyz"},
		"a GitHub token":   {unmask("{{ghp_}}abcdefghijklmnopqrstuvwxyz0123456789"), "abcdefghijklmnopqrstuvwxyz0123456789"},
		"an AWS key id":    {awsKeyID, awsKeyID},
		"a Google key":     {unmask("{{AIza}}SyA-abcdefghijklmnopqrstuvwxyz012345"), "abcdefghijklmnopqrstuvwxyz012345"},
		"a Slack token":    {unmask("{{xoxb-}}1234567890-abcdefghij"), "1234567890-abcdefghij"},
		"a bearer token":   {"Bearer abcdef.ghijkl-mnop", "abcdef.ghijkl-mnop"},
		"a URL password":   {"postgres://admin:s3cretpass@db.example.com/app", "s3cretpass"},
		"a named password": {"DB_PASS=hunter2hunter2", "hunter2hunter2"},
		"an encoded value": {
			"ZmFrZXNlY3JldHZhbHVlMTIzNDU2Nzg5MGFiY2RlZg9z", "ZmFrZXNlY3JldHZhbHVlMTIzNDU2Nzg5MGFiY2RlZg9z",
		},
	}

	for shape, pair := range shapes {
		text, needle := pair[0], pair[1]

		t.Run(shape, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Model = repliesInOrder(lessonsText(lessonWith(nil)))

			world.Do(t, captureOf([]any{
				said("user", "No, use targ. Here is the config: "+text), said("assistant", "Understood."),
			}))

			g.Expect(world.Requests).NotTo(BeEmpty())
			g.Expect(callsOf(world)).NotTo(ContainSubstring(needle))
			g.Expect(world.Find("learn")).To(HaveLen(1))
		})
	}
}

func Test793d26b1_ASecretThatEvadedASinglePatternDoesNotReachTheJudgement(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	long := "No, use targ. " + strings.Repeat("x ", 590) + "sk-ant-api03-SECRETSPLITATEDGE0123456789"
	secrets := []string{
		"SECRETSPLITATEDGE0123456789",
		"WRAPPEDKEYPARTONE1234567890ab\nWRAPPEDKEYPARTTWO1234567890cd",
		"hunter two with spaces",
		"%73%6B%2D%61%6E%74%2DENCODEDSECRET0123456789ABCDEF",
		"s3cretpassinurl",
		unmask("{{eyJ}}hbGciOiJIUzI1NiJ9.{{eyJ}}zdWIiOiJKV1RQQVlMT0FEIn0.JWTSIGNATURE1234567890"),
	}
	text := long + "\nkey: " + secrets[1] + "\n{\"password\": \"" + secrets[2] + "\"}\nsee " + secrets[3] +
		"\npostgres://bob:" + secrets[4] + "@db/x\n" + secrets[5]

	world.Do(t, captureOf([]any{said("user", text)}))

	for _, secret := range secrets {
		for part := range strings.SplitSeq(secret, "\n") {
			g.Expect(world.Requests[0].Prompt).NotTo(ContainSubstring(part), part)
		}
	}

	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("No, use targ."))
}

func Test793d26b1_RedactNeverReturnsATextHoldingAKnownSecretAndIsIdempotentOnItsOutput(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		prose := rapid.StringMatching(`[a-z ,.]{0,60}`).Draw(rt, "prose")
		body := rapid.StringMatching(`[A-Za-z0-9]{20,40}`).Draw(rt, "body")
		prefixes := []string{"sk-ant-", "ghp_", "xoxb-", "Bearer ", "password=", "api_key: "}
		prefix := rapid.SampledFrom(prefixes).Draw(rt, "prefix")
		text := prose + " " + prefix + body + " " + prose

		once := auto.Redact(text)

		if strings.Contains(once, body) {
			rt.Fatalf("the secret %q survived in %q", body, once)
		}

		if again := auto.Redact(once); again != once {
			rt.Fatalf("redaction is not idempotent: %q then %q", once, again)
		}
	})
}

// The time of the redaction grows in proportion to the text (793d26b1): four times the text costs about four times as
// long, where the quadratic redactor of the JavaScript twin cost sixteen times as long. A ratio holds on a slow or busy
// machine where a wall-clock budget does not.
func Test793d26b1_RedactionCostsTimeInProportionToTheText(t *testing.T) {
	t.Parallel()

	texts := map[string][2]string{
		"an unbroken run": {strings.Repeat("a", unbrokenRun), strings.Repeat("a", unbrokenRun*growth)},
		"a block of wrapped lines": {
			strings.Repeat("abcdefghijklmnopqrstuvwxyz0123\n", wrappedBlockRows),
			strings.Repeat("abcdefghijklmnopqrstuvwxyz0123\n", wrappedBlockRows*growth),
		},
	}

	for name, pair := range texts {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			small, large := redactionCost(t, pair[0]), redactionCost(t, pair[1])

			g.Expect(large).To(BeNumerically("<=", small*proportionSlack+redactSlack))
		})
	}
}

func Test793d26b1_SecretsOfTheShapesAReviewProbedDoNotReachTheJudgement(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	secrets := []string{
		"OPENAIVALUE123", "dbhunter2value", "passhunter2value", "abc.def", "headerpart.second", "trailing words",
		"OPENAIVALUE1232",
	}
	lines := []string{
		"No, use targ.",
		"OPENAI_KEY=" + secrets[0],
		"DB_PASS=" + secrets[1],
		"pass: " + secrets[2],
		`curl -H "Authorization: Bearer ` + secrets[3] + `"`,
		"Authorization: Bearer " + secrets[4] + " " + secrets[5],
		"MY_SERVICE_KEY: " + secrets[6],
	}
	messages := make([]any, 0, len(lines))

	for index, line := range lines {
		role := "assistant"
		if index == 0 {
			role = "user"
		}

		messages = append(messages, said(role, line))
	}

	world.Do(t, captureOf(messages))

	for _, secret := range secrets {
		g.Expect(world.Requests[0].Prompt).NotTo(ContainSubstring(secret), secret)
	}

	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("No, use targ."))
}

func Test793d27b1_AWindowNeverExceedsTheWindowLimit(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		count := rapid.IntRange(1, 30).Draw(rt, "count")
		messages := make([]auto.Message, 0, count)

		for range count {
			role := rapid.SampledFrom([]string{"user", "assistant"}).Draw(rt, "role")
			text := rapid.StringN(0, 2500, -1).Draw(rt, "text")
			messages = append(messages, auto.Message{Role: role, Text: text})
		}

		window := auto.WindowFromMessages(messages, rapid.String().Draw(rt, "answer"))

		if length := len([]rune(window)); length > windowCharLimit {
			rt.Fatalf("a window of %d characters", length)
		}

		for _, one := range auto.WindowsFromMessage(messages) {
			if length := len([]rune(one)); length > windowCharLimit+windowCharLimit/2 {
				rt.Fatalf("a backlog window of %d characters", length)
			}
		}
	})
}

func Test793e8g_ACapturedTurnMarksTheTurnsOfItsSessionByTheNameOfTheFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Model = repliesInOrder("[]")
	job := captureOf(turnMessages())
	job["transcript"] = transcriptPath

	world.Do(t, job)

	turns, written := marksOf(t, world, defaultChunks)
	g.Expect(written).To(BeTrue())
	g.Expect(turns).To(HaveKeyWithValue("9f1e2d3c.jsonl", float64(world.Clock.UnixMilli())))
	g.Expect(world.Writes).To(Equal([]string{defaultChunks + marksFile}))
}

func Test793e8g_AMarkThatCannotBeWrittenGoesToTheDebugLogOnlyAndTheUserIsNotWarned(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Model = repliesInOrder("[]")
	world.Files[defaultChunks+marksFile] = "not json"
	job := captureOf(turnMessages())
	job["transcript"] = transcriptPath

	world.Do(t, job)

	g.Expect(world.Warnings).To(BeEmpty())
	g.Expect(world.Writes).To(BeEmpty())
	g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("internal:"))
}

func Test793e8g_ATurnIsNotMarkedWhenNoTranscriptIsNamedOrTheJudgementFailed(t *testing.T) {
	t.Parallel()

	t.Run("no transcript", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		world := NewWorld()
		world.Model = repliesInOrder("[]")

		world.Do(t, captureOf(turnMessages()))

		g.Expect(world.Writes).To(BeEmpty())
	})

	t.Run("a failed judgement", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		world := NewWorld()
		world.Model = func(string, auto.Request) map[string]any { return Unanswered("api-error") }
		job := captureOf(turnMessages())
		job["transcript"] = transcriptPath

		world.Do(t, job)

		g.Expect(world.Writes).To(BeEmpty())
	})
}

func Test793e8g_TheChunksDirectoryFollowsTheVariablesOfTheEngramCommands(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		env map[string]string
		dir string
	}{
		"ENGRAM_CHUNKS_DIR": {map[string]string{"ENGRAM_CHUNKS_DIR": "/custom/chunks"}, "/custom/chunks"},
		"XDG_DATA_HOME":     {map[string]string{"XDG_DATA_HOME": "/xdg"}, "/xdg/engram/chunks"},
		"the home, no variable": {
			map[string]string{"XDG_DATA_HOME": "", "HOME": "/home/me"}, "/home/me/.local/share/engram/chunks",
		},
	}

	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Model = repliesInOrder("[]")

			stdmaps.Copy(world.Env, one.env)

			job := captureOf(turnMessages())
			job["transcript"] = transcriptPath

			world.Do(t, job)

			g.Expect(world.Writes).To(Equal([]string{one.dir + marksFile}))
		})
	}
}

func Test793e8g_TheMarksOfAnEarlierRunAreKeptAndTheFirstRunFixesTheCutoffOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Model = repliesInOrder("[]")
	world.Files[defaultChunks+marksFile] = `{"sources":{"/a.jsonl":"sha256:a"},"turns":{"old.jsonl":5},"partial":{},` +
		`"installed":10,"cutoff":3}`
	job := captureOf(turnMessages())
	job["transcript"] = transcriptPath

	world.Do(t, job)

	var marks marksJSON

	g.Expect(json.Unmarshal([]byte(world.Written[defaultChunks+marksFile]), &marks)).To(Succeed())
	g.Expect(marks.Sources).To(HaveKeyWithValue("/a.jsonl", "sha256:a"))
	g.Expect(marks.Turns).To(HaveKeyWithValue("old.jsonl", float64(5)))
	g.Expect(marks.Turns).To(HaveKey("9f1e2d3c.jsonl"))
	g.Expect(marks.Cutoff).To(Equal(float64(3)))
}

// unexported constants.
const (
	awsKeyTail       = 16
	costRuns         = 3
	defaultChunks    = "/home/t/.local/share/engram/chunks"
	growth           = 4
	marksFile        = "/backlog-marks.json"
	proportionSlack  = 8
	redactSlack      = 20 * time.Millisecond
	transcriptPath   = "/private/home/.claude/projects/p/9f1e2d3c.jsonl"
	unbrokenRun      = 12_000
	windowCharLimit  = 8000
	wrappedBlockRows = 500
)

// marksJSON is the mark file of the backlog as the tests read it.
type marksJSON struct {
	Sources map[string]string  `json:"sources"`
	Turns   map[string]float64 `json:"turns"`
	Cutoff  float64            `json:"cutoff"`
}

// callsOf is everything the job sent out: the prompts of the model and the command lines.
func callsOf(world *World) string {
	seen := make([]string, 0, len(world.Requests)+len(world.Commands))

	for _, request := range world.Requests {
		seen = append(seen, request.Prompt)
	}

	for _, command := range world.Commands {
		seen = append(seen, command.Line())
	}

	return strings.Join(seen, "\n")
}

// marksOf reads the turns marked in the mark file the job wrote.
func marksOf(t *testing.T, world *World, dir string) (map[string]float64, bool) {
	t.Helper()

	text, written := world.Written[dir+marksFile]
	if !written {
		return nil, false
	}

	var marks marksJSON

	err := json.Unmarshal([]byte(text), &marks)
	if err != nil {
		t.Fatalf("the marks cannot be read: %v", err)
	}

	return marks.Turns, true
}

// redactionCost is the least time of three captures of a message of this text, the redaction being their only cost.
func redactionCost(t *testing.T, text string) time.Duration {
	t.Helper()

	least := time.Hour

	for range costRuns {
		world := NewWorld()
		start := time.Now()

		world.Do(t, captureOf([]any{said("user", text)}))

		least = min(least, time.Since(start))
	}

	return least
}
