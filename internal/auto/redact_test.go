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

// credential is a text shaped as a credential, the secret in it, and the name of an assignment that is kept.
type credential struct {
	text   string
	secret string
	name   string
}

// credentialShape generates a text of one of the shapes of a credential, built here and never written whole in this
// file: a scanner of the repository would take a literal for a credential.
func credentialShape() *rapid.Generator[credential] {
	word := func(alphabet string, low, high int) *rapid.Generator[string] {
		return rapid.StringMatching(`[` + alphabet + `]{` + strconv.Itoa(low) + `,` + strconv.Itoa(high) + `}`)
	}
	alnum := "A-Za-z0-9"
	whole := func(secret string) credential { return credential{text: secret, secret: secret} }

	return rapid.OneOf(
		rapid.Map(word("0-9A-Z", 16, 16), func(body string) credential { return whole("AK" + "IA" + body) }),
		rapid.Map(word("0-9A-Z", 16, 16), func(body string) credential { return whole("AS" + "IA" + body) }),
		rapid.Map(rapid.Tuple2(rapid.SampledFrom([]string{"gh" + "p_", "gh" + "o_", "gh" + "s_", "gh" + "u_", "gh" + "r_",
			"github" + "_pat_", "np" + "m_", "gl" + "pat-"}), word(alnum+"_", 20, 40)),
			func(parts rapid.Tuple2[string, string]) credential { return whole(parts.A + parts.B) }),
		rapid.Map(rapid.Tuple2(rapid.SampledFrom([]string{"s" + "k", "p" + "k", "r" + "k"}), word(alnum+"_-", 20, 40)),
			func(parts rapid.Tuple2[string, string]) credential { return whole(parts.A + "-" + parts.B) }),
		rapid.Map(word(alnum+"_-", 30, 40), func(body string) credential { return whole("AI" + "za" + body) }),
		rapid.Map(rapid.Tuple2(rapid.SampledFrom([]string{"a", "b", "p", "r", "s"}), word(alnum+"-", 10, 30)),
			func(parts rapid.Tuple2[string, string]) credential {
				return whole("xo" + "x" + parts.A + "-" + parts.B)
			}),
		rapid.Map(rapid.Tuple3(word(alnum+"_-", 8, 20), word(alnum+"_-", 8, 20), word(alnum+"_-", 8, 20)),
			func(parts rapid.Tuple3[string, string, string]) credential {
				return whole("ey" + "J" + parts.A + "." + parts.B + "." + parts.C)
			}),
		rapid.Map(word(alnum+"._~+/=-", 8, 30), func(body string) credential {
			return credential{text: "Bearer " + body, secret: body}
		}),
		rapid.Map(word(alnum+"._~+/=-", 12, 30), func(body string) credential {
			return credential{text: "Basic " + body, secret: body}
		}),
		rapid.Map(word("a-zA-Z0-9+/= \n", 20, 60), func(body string) credential {
			block := "-----BEGIN RSA PRIVATE KEY-----\n" + body + "\n-----END RSA PRIVATE KEY-----"

			return credential{text: block, secret: strings.TrimSpace(body)}
		}),
		rapid.Map(rapid.Tuple2(word("a-z", 3, 8), word(alnum+"!#%^*_+-", 6, 20)),
			func(parts rapid.Tuple2[string, string]) credential {
				return credential{text: "https://" + parts.A + ":" + parts.B + "@example.test/path", secret: parts.B}
			}),
		rapid.Map(word(alnum+"._~+/=-", 8, 30), func(value string) credential {
			return credential{text: "Authorization: Token " + value, secret: value, name: "Authorization"}
		}),
		rapid.Map(rapid.Tuple3(rapid.SampledFrom([]string{"password", "passwd", "pwd", "api_key", "secret", "token",
			"my_secret_token", "client_credential", "private_key", "AWS_SECRET_ACCESS_KEY"}),
			rapid.SampledFrom([]string{"=", ": ", " is "}), word(alnum+"!#%^*_+-", 6, 20)),
			func(parts rapid.Tuple3[string, string, string]) credential {
				return credential{text: parts.A + parts.B + parts.C, secret: parts.C, name: parts.A}
			}),
		rapid.Map(rapid.Tuple2(word("a-z", 8, 20), word("0-9", 12, 20)), func(parts rapid.Tuple2[string, string]) credential {
			return whole(parts.A + parts.B + "Zq")
		}),
	)
}
