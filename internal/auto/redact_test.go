package auto_test

import (
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4j1a_Redact_NoTextShapedAsACredentialSurvives(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		found := credentialShape().Draw(rt, "credential")
		before := rapid.StringMatching(`[a-zA-Z .,]{0,30}`).Draw(rt, "before")
		after := rapid.StringMatching(`[a-zA-Z .,]{0,30}`).Draw(rt, "after")

		redacted := auto.Redact(before + " " + found.text + " " + after)

		if strings.Contains(redacted, found.secret) {
			rt.Fatalf("the secret %q survived in %q", found.secret, redacted)
		}

		if found.name != "" && !strings.Contains(redacted, found.name) {
			rt.Fatalf("the name %q of the assignment is not in %q", found.name, redacted)
		}
	})
}

func Test793d4j1b_Redact_TheNameOfAnAssignmentIsKeptAndWhatIsNotACredentialIsNotTouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(auto.Redact("the password: hunter2hunter2 was set")).To(Equal("the password: [redacted] was set"))
	g.Expect(auto.Redact("export api_key=abc123def456 now")).To(Equal("export api_key=[redacted] now"))
	g.Expect(auto.Redact("SECRET_TOKEN is s3cr3t-value")).To(Equal("SECRET_TOKEN is [redacted]"))
	g.Expect(auto.Redact(`token = "a quoted value" end`)).To(Equal(`token = [redacted] end`))

	for _, kept := range []string{
		"9fceb02d0ae598e95dc970b74767f19372d61af8",
		strings.Repeat("supercalifragilistic", 3),
		"Run the formatter before committing a Python file.",
		"build with make zorblax-build, never with make all",
	} {
		g.Expect(auto.Redact(kept)).To(Equal(kept))
	}
}

// unexported constants.
const (
	alnum = "A-Za-z0-9"
	// the shapes credentialShape draws: tokens of an issuer up to lastToken, texts of a header up to lastHeader, and
	// values of a name and long runs after them
	lastHeader = 10
	lastShape  = 13
	lastToken  = 5
)

// credential is a text shaped as a credential, the secret in it, and the name of an assignment that is kept.
type credential struct {
	text   string
	secret string
	name   string
}

// credentialShape generates a text of one of the shapes of a credential, built here and never written whole in this
// file: a scanner of the repository would take a literal for a credential.
func credentialShape() *rapid.Generator[credential] {
	return rapid.Custom(func(rt *rapid.T) credential {
		switch shape := rapid.IntRange(0, lastShape).Draw(rt, "shape"); {
		case shape <= lastToken:
			return issuedToken(rt, shape)
		case shape <= lastHeader:
			return headerCredential(rt, shape)
		default:
			return namedValue(rt, shape)
		}
	})
}

// headerCredential is a Bearer or Basic credential, a private key block, the user and password of a URL, or the value
// of an Authorization header.
func headerCredential(rt *rapid.T, shape int) credential {
	switch shape {
	case 6:
		body := word(rt, "body", alnum+"._~+/=-", 8, 30)

		return credential{text: "Bearer " + body, secret: body}
	case 7:
		body := word(rt, "body", alnum+"._~+/=-", 12, 30)

		return credential{text: "Basic " + body, secret: body}
	case 8:
		body := word(rt, "body", "a-zA-Z0-9+/= \n", 20, 60)

		return credential{
			text:   "-----BEGIN RSA PRIVATE KEY-----\n" + body + "\n-----END RSA PRIVATE KEY-----",
			secret: strings.TrimSpace(body),
		}
	case 9:
		password := word(rt, "password", alnum+"!#%^*_+-", 6, 20)
		user := word(rt, "user", "a-z", 3, 8)

		return credential{text: "https://" + user + ":" + password + "@example.test/path", secret: password}
	default:
		value := word(rt, "value", alnum+"._~+/=-", 8, 30)

		return credential{text: "Authorization: Token " + value, secret: value, name: "Authorization"}
	}
}

// issuedToken is a key or token that opens with the name of its issuer, or a JSON web token.
func issuedToken(rt *rapid.T, shape int) credential {
	var secret string

	switch shape {
	case 0:
		issuer := rapid.SampledFrom([]string{"AK" + "IA", "AS" + "IA"}).Draw(rt, "issuer")
		secret = issuer + word(rt, "body", "0-9A-Z", 16, 16)
	case 1:
		prefix := rapid.SampledFrom([]string{"gh" + "p_", "gh" + "o_", "gh" + "s_", "gh" + "u_", "gh" + "r_",
			"github" + "_pat_", "np" + "m_", "gl" + "pat-"}).Draw(rt, "prefix")
		secret = prefix + word(rt, "body", alnum+"_", 20, 40)
	case 2:
		secret = rapid.SampledFrom([]string{"s" + "k", "p" + "k", "r" + "k"}).Draw(rt, "issuer") + "-" +
			word(rt, "body", alnum+"_-", 20, 40)
	case 3:
		secret = "AI" + "za" + word(rt, "body", alnum+"_-", 30, 40)
	case 4:
		secret = "xo" + "x" + rapid.SampledFrom([]string{"a", "b", "p", "r", "s"}).Draw(rt, "kind") + "-" +
			word(rt, "body", alnum+"-", 10, 30)
	default:
		secret = "ey" + "J" + word(rt, "head", alnum+"_-", 8, 20) + "." + word(rt, "body", alnum+"_-", 8, 20) + "." +
			word(rt, "tail", alnum+"_-", 8, 20)
	}

	return credential{text: secret, secret: secret}
}

// namedValue is a value given to a name that holds a word of a secret, or a long run of letters and digits.
func namedValue(rt *rapid.T, shape int) credential {
	if shape == lastShape {
		secret := word(rt, "letters", "a-z", 16, 30) + word(rt, "digits", "0-9", 16, 30) + "Zq"

		return credential{text: secret, secret: secret}
	}

	name := rapid.SampledFrom([]string{"password", "passwd", "pwd", "api_key", "secret", "token", "my_secret_token",
		"client_credential", "private_key", "AWS_SECRET_ACCESS_KEY"}).Draw(rt, "name")
	separator := rapid.SampledFrom([]string{"=", ": ", " is "}).Draw(rt, "separator")
	value := word(rt, "value", alnum+"!#%^*_+-", 6, 20)

	return credential{text: name + separator + value, secret: value, name: name}
}

// word draws a text of the characters of an alphabet, as long as a range says.
func word(rt *rapid.T, label, alphabet string, low, high int) string {
	return rapid.StringMatching(`[`+alphabet+`]{`+strconv.Itoa(low)+`,`+strconv.Itoa(high)+`}`).Draw(rt, label)
}
