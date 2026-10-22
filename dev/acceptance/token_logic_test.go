package acceptance_test

import (
	"errors"
	"regexp"
	"strings"
)

// unexported constants.
const (
	// credentialFile is the owner's credential file, relative to the home directory.
	credentialFile = ".config/claude-key.env"
	// tokenVariable is the environment variable, and the variable of the credential file, that hold the login.
	tokenVariable = "CLAUDE_CODE_OAUTH_TOKEN"
)

// unexported variables.
var (
	errNoToken   = errors.New("no credentials")
	tokenPattern = regexp.MustCompile(`^\s*(?:export\s+)?` + tokenVariable + `=(.*?)\s*$`)
)

// skipper is the part of a test that skips it: the real test ends, a recorder only records.
type skipper interface {
	Helper()
	Skipf(format string, args ...any)
}

// lookupToken returns the login from the environment or, failing that, from the credential file read through
// readCredentials, and whether there is one. The environment wins.
func lookupToken(getenv func(string) string, readCredentials func() ([]byte, error)) (string, bool) {
	token := getenv(tokenVariable)
	if token != "" {
		return token, true
	}

	data, err := readCredentials()
	if err != nil {
		return "", false
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		match := tokenPattern.FindStringSubmatch(line)
		if match != nil {
			return strings.Trim(match[1], `"'`), true
		}
	}

	return "", false
}

// requireToken returns the login the live tests use, and skips the test when there is none, with a message that
// names the variable and the credential file that supply it (862b2b). It is handed to the child's environment only,
// never to a command line or a log.
func requireToken(t skipper, getenv func(string) string, readCredentials func() ([]byte, error)) string {
	t.Helper()

	token, found := lookupToken(getenv, readCredentials)
	if !found {
		t.Skipf("%v: set %s, or put it in ~/%s", errNoToken, tokenVariable, credentialFile)

		return ""
	}

	return token
}
