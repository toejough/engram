package acceptance_test

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

// Test862a3b1a_RequireToken_ACheckWithoutALoginSkipsAndSaysWhichLogin verifies 862a3b1a: when no login exists, the
// check that needs one is skipped and the skip says which login it needs.
func Test862a3b1a_RequireToken_ACheckWithoutALoginSkipsAndSaysWhichLogin(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	recorder := &skipRecorder{}

	token := requireToken(recorder, environmentOf(nil), credentialFileWith(""))

	g.Expect(recorder.skipped).To(BeTrue(), "the check was not skipped")
	g.Expect(recorder.message).To(ContainSubstring(tokenVariable))
	g.Expect(token).To(BeEmpty())
}

// Test862b2b1a_RequireToken_AMissingLoginSkipsTheTestAndNamesTheVariableAndTheFile verifies 862b2b1a: neither the
// token variable nor the credential file supplies a login, so the test is skipped with a message that names both.
func Test862b2b1a_RequireToken_AMissingLoginSkipsTheTestAndNamesTheVariableAndTheFile(t *testing.T) {
	t.Parallel()

	for name, scenario := range map[string]tokenScenario{
		"no variable and an empty credential file": {
			getenv: environmentOf(nil), readCredentials: credentialFileWith(""),
		},
		"an empty variable": {
			getenv: environmentOf(map[string]string{tokenVariable: ""}), readCredentials: credentialFileWith(""),
		},
		"no credential file": {
			getenv: environmentOf(nil), readCredentials: unreadableCredentialFile,
		},
		"a credential file that holds another variable": {
			getenv: environmentOf(nil), readCredentials: credentialFileWith("OTHER_VARIABLE=abc\n# a comment\n"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			recorder := &skipRecorder{}

			token := requireToken(recorder, scenario.getenv, scenario.readCredentials)

			g.Expect(recorder.skipped).To(BeTrue(), "the test was not skipped")
			g.Expect(recorder.message).To(ContainSubstring(tokenVariable))
			g.Expect(recorder.message).To(ContainSubstring("~/" + credentialFile))
			g.Expect(token).To(BeEmpty())
		})
	}
}

// Test862b2b1b_RequireToken_ALoginThatExistsIsUsed verifies 862b2b1b: with the token variable set the test continues
// with that token and is not skipped; the credential file supplies the login when the variable does not.
func Test862b2b1b_RequireToken_ALoginThatExistsIsUsed(t *testing.T) {
	t.Parallel()

	for name, scenario := range map[string]tokenScenario{
		"the variable": {
			getenv: environmentOf(map[string]string{tokenVariable: wantedToken}), readCredentials: unreadableCredentialFile,
			want: wantedToken,
		},
		"the variable wins over the credential file": {
			getenv:          environmentOf(map[string]string{tokenVariable: wantedToken}),
			readCredentials: credentialFileWith(tokenVariable + "=from-the-file\n"), want: wantedToken,
		},
		"the credential file": {
			getenv:          environmentOf(nil),
			readCredentials: credentialFileWith("# the login\nexport " + tokenVariable + "=\"" + wantedToken + "\"\n"),
			want:            wantedToken,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			recorder := &skipRecorder{}

			token := requireToken(recorder, scenario.getenv, scenario.readCredentials)

			g.Expect(recorder.skipped).To(BeFalse(), "the test was skipped: "+recorder.message)
			g.Expect(token).To(Equal(scenario.want))
		})
	}
}

// unexported constants.
const (
	wantedToken = "token-for-the-test"
)

// unexported variables.
var (
	errNoCredentialFile = errors.New("no such credential file")
)

// skipRecorder is a test that records its skip instead of ending.
type skipRecorder struct {
	message string
	skipped bool
}

func (recorder *skipRecorder) Helper() {}

func (recorder *skipRecorder) Skipf(format string, args ...any) {
	recorder.skipped = true
	recorder.message = fmt.Sprintf(format, args...)
}

// tokenScenario is the environment and the credential file one test of the login lookup runs against.
type tokenScenario struct {
	getenv          func(string) string
	readCredentials func() ([]byte, error)
	want            string
}

// credentialFileWith returns a reader of a credential file that holds the text.
func credentialFileWith(text string) func() ([]byte, error) {
	return func() ([]byte, error) { return []byte(text), nil }
}

// environmentOf returns a lookup of the variables, which finds none other.
func environmentOf(variables map[string]string) func(string) string {
	return func(name string) string { return variables[name] }
}

// unreadableCredentialFile is the reader of a credential file that does not exist.
func unreadableCredentialFile() ([]byte, error) {
	return nil, errNoCredentialFile
}
