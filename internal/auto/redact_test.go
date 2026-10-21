package auto_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4j1a_Redact_NoTextShapedAsACredentialSurvives(t *testing.T) {
	t.Parallel()

	for _, spec := range credentialSpecs() {
		t.Run(spec.name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				found := spec.make(rt)
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
		})
	}
}

func Test793d4j1b_Redact_TheNameOfAnAssignmentIsKeptAndWhatIsNotACredentialIsNotTouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(auto.Redact("the password: hunter2hunter2 was set")).To(Equal("the password: [redacted] was set"))
	g.Expect(auto.Redact("export api_key=abc123def456 now")).To(Equal("export api_key=[redacted] now"))
	g.Expect(auto.Redact("SECRET_TOKEN is s3cr3t-value")).To(Equal("SECRET_TOKEN is [redacted]"))
	g.Expect(auto.Redact(`token = "a quoted value" end`)).To(Equal(`token = [redacted] end`))

	// the forms that the promise leaves out, and the shapes one character short of their lower bound
	for _, kept := range []string{
		"9fceb02d0ae598e95dc970b74767f19372d61af8",
		strings.Repeat("supercalifragilistic", 3),
		"Run the formatter before committing a Python file.",
		"build with make zorblax-build, never with make all",
		strings.Repeat("a", 36),
		strings.Repeat("7", 36),
		"abcdefgh.ijklmnop.qrstuvwx",
		strings.Repeat("a1", 15),
		"s" + "k-" + strings.Repeat("a", 15),
		"AK" + "IA" + strings.Repeat("A", 15),
		"AK" + "IA" + strings.Repeat("A", 17),
		"AI" + "za" + strings.Repeat("a", 29),
		"xo" + "xb-" + strings.Repeat("a", 9),
		"ey" + "J" + strings.Repeat("a", 7) + ".abcdefgh.abcdefgh",
	} {
		g.Expect(auto.Redact(kept)).To(Equal(kept))
	}
}

func Test793d4j1e_Redact_TheValueOfEveryNameThatHoldsAWordOfASecretIsRedactedWhateverTheNameLooksLike(t *testing.T) {
	t.Parallel()

	t.Run("the names that the pattern missed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// built here and never written whole in this file: a scanner of the repository would take a literal for a key
		value := "wJalr" + "XUtnFEMI/K7MDENG"

		g.Expect(auto.Redact("key=hunter2hunter2")).To(Equal("key=[redacted]"))
		g.Expect(auto.Redact("accessKey: " + value)).To(Equal("accessKey: [redacted]"))
		g.Expect(auto.Redact("signingKey=abcd1234efgh")).To(Equal("signingKey=[redacted]"))
		g.Expect(auto.Redact("x-api-key: " + value)).To(Equal("x-api-key: [redacted]"))
		g.Expect(auto.Redact("DB_PASSWORD is hunter2hunter2")).To(Equal("DB_PASSWORD is [redacted]"))
	})

	t.Run("any name of the words, in any case, with a word before and after", func(t *testing.T) {
		t.Parallel()

		rapid.Check(t, func(rt *rapid.T) {
			name := secretName(rt)
			separator := rapid.SampledFrom([]string{"=", ": ", " is "}).Draw(rt, "separator")
			value := word(rt, "value", alnum+"!#%^*_+-", 6, 20)

			redacted := auto.Redact(name + separator + value)

			if strings.Contains(redacted, value) {
				rt.Fatalf("the value %q survived in %q", value, redacted)
			}

			if !strings.HasPrefix(redacted, name+separator) {
				rt.Fatalf("the name and the separator %q are not kept in %q", name+separator, redacted)
			}
		})
	})
}

func Test793d4j1f_Redact_ASentenceGivesTheValueOfANameToRedact(t *testing.T) {
	t.Parallel()

	t.Run("the sentence of review finding 3", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(auto.Redact("the password for the staging deploy is hunter2hunter2")).
			To(Equal("the password for the staging deploy is [redacted]"))
	})

	t.Run("up to six words, and the text as it was with seven", func(t *testing.T) {
		t.Parallel()

		rapid.Check(t, func(rt *rapid.T) {
			name := secretName(rt)
			words := rapid.SliceOfN(rapid.SampledFrom(fillerWords), 0, maxWordsBetween+1).Draw(rt, "words")
			separator := rapid.SampledFrom([]string{" is ", ": "}).Draw(rt, "separator")
			value := word(rt, "value", alnum+"!#%^*_+-", 6, 20)

			before := "the " + name
			if len(words) > 0 {
				before += " " + strings.Join(words, " ")
			}

			text := before + separator + value
			redacted := auto.Redact(text)

			if len(words) > maxWordsBetween {
				if redacted != text {
					rt.Fatalf("a sentence of %d words between the name and the separator was changed: %q", len(words), redacted)
				}

				return
			}

			if strings.Contains(redacted, value) {
				rt.Fatalf("the value %q survived in %q", value, redacted)
			}

			if !strings.HasPrefix(redacted, before+separator) {
				rt.Fatalf("the name and the words %q are not kept in %q", before+separator, redacted)
			}
		})
	})
}

func Test793d4j1g_Redact_ABearerCredentialIsRedactedFrom12Characters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(auto.Redact("Bearer " + strings.Repeat("a", 11))).To(Equal("Bearer " + strings.Repeat("a", 11)))
	g.Expect(auto.Redact("Bearer " + strings.Repeat("a", 12))).To(Equal("[redacted]"))
	g.Expect(auto.Redact("Authentication is by Bearer token, Basic is for tools")).
		To(Equal("Authentication is by Bearer token, Basic is for tools"))
}

func Test793d4j1h_Judge_ACredentialThatCharactersNotShownSplitIsNotInTheRequest(t *testing.T) {
	t.Parallel()

	t.Run("the splits of the review", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// built here and never written whole in this file: a scanner of the repository would take a literal for a key
		key := "AK" + "IA" + "IOSFODNN7EXAMPLE"
		token := "gh" + "p_" + "aZ3aZ3aZ3a" + "Z3aZ3aZ3aZ3aZ3aZ3aZ3aZ3"
		password := "hunter" + "2hunter2"

		for name, text := range map[string]string{
			"a key split by a zero-width space":  "use " + key[:6] + "\u200b" + key[6:] + " for the deploy",
			"a token split by a soft hyphen":     "use " + token[:12] + "\u00ad" + token[12:] + " for the deploy",
			"a token split by a control char":    "use " + token[:12] + "\x00" + token[12:] + " for the deploy",
			"a name split so that it is no name": "the pass\u200bword: " + password,
		} {
			request, found := judge(t, userPrompt(jsonText(text)), "ok", nil)

			g.Expect(found).To(BeTrue(), name)
			g.Expect(request.Prompt).NotTo(ContainSubstring(key), name)
			g.Expect(request.Prompt).NotTo(ContainSubstring(token), name)
			g.Expect(request.Prompt).NotTo(ContainSubstring(password), name)
		}
	})

	t.Run("a line break is white space to the redaction", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// the value on the line after its name is redacted, and a header takes the rest of its line and no more
		password := "hunter" + "2hunter2"
		header := "Authorization: Bearer " + strings.Repeat("tok3n", 4)

		request, found := judge(t, userPrompt(jsonText("the password:\n"+password+"\n"+header+"\nthen I said hello")),
			"ok", nil)

		g.Expect(found).To(BeTrue())
		g.Expect(request.Prompt).NotTo(ContainSubstring(password))
		g.Expect(request.Prompt).NotTo(ContainSubstring("tok3n"))
		g.Expect(request.Prompt).To(ContainSubstring("then I said hello"), "the text after the header is not redacted")
	})

	t.Run("any credential, any split, in a message, an answer and a note", func(t *testing.T) {
		t.Parallel()

		rapid.Check(t, func(rt *rapid.T) {
			inMessage := splitCredential().Draw(rt, "in a message")
			inAnswer := splitCredential().Draw(rt, "in an answer")
			inNote := splitCredential().Draw(rt, "in a note")
			record := userPrompt(jsonText("No, use the script. "+inMessage.text+" and again")) + "\n" +
				assistantText(jsonText("Understood, "+inAnswer.text))

			request, found := judge(rt, record, "the answer "+inAnswer.text,
				[]auto.Note{{Path: "1.md", Situation: "s", Body: "note " + inNote.text}})
			if !found {
				rt.Fatalf("a record with a prompt was judged nothing")
			}

			for _, secret := range []string{inMessage.secret, inAnswer.secret, inNote.secret} {
				if strings.Contains(request.Prompt, secret) {
					rt.Fatalf("the request holds the secret %q: %q", secret, request.Prompt)
				}
			}
		})
	})
}

// unexported constants.
const (
	alnum = "A-Za-z0-9"
	// boundary is how many characters above its lower bound a part of a shape may run: the lower bound is drawn often,
	// so that the bound is probed.
	boundary = 5
	// maxWordsBetween is how many words of a sentence may stand between a name and the is or colon that gives its value.
	maxWordsBetween = 6
)

// unexported variables.
var (
	fillerWords = []string{"for", "the", "staging", "deploy", "of", "my", "new", "old", "admin", "server", "main",
		"account"}
	// notShown are the characters and the white space that can be put inside a credential: a zero-width space and
	// joiner, a soft hyphen, a byte-order mark, a bidirectional override and isolate, control characters, a tab and a
	// space.
	notShown = []string{"\u200b", "\u200d", "\u00ad", "\ufeff", "\u202e", "\u2066", "\x00", "\x1b", "\x7f", "\t", " "}
)

// credential is a text shaped as a credential, the secret in it, and the name of an assignment that is kept.
type credential struct {
	text   string
	secret string
	name   string
}

// credentialSpec is one shape of the recognised credentials, with a generator of texts of the shape whose
// parts run up from their lower bound.
type credentialSpec struct {
	name string
	make func(rt *rapid.T) credential
}

// authorizationValue is the value of an Authorization header.
func authorizationValue(rt *rapid.T) credential {
	value := word(rt, "value", alnum+"._~+/=-", 8, 30)

	return credential{text: "Authorization: Token " + value, secret: value, name: "Authorization"}
}

// credentialShape generates a text of one of the shapes of a credential.
func credentialShape() *rapid.Generator[credential] {
	specs := credentialSpecs()

	return rapid.Custom(func(rt *rapid.T) credential {
		return rapid.SampledFrom(specs).Draw(rt, "shape").make(rt)
	})
}

// credentialSpecs lists the shapes of a credential that the promise names, one for each prefix of an
// issuer, built here and never written whole in this file: a scanner of the repository would take a literal for a
// credential.
func credentialSpecs() []credentialSpec {
	specs := []credentialSpec{
		{name: "a private key block", make: privateKeyBlock},
		{name: "a token of three parts that opens with eyJ", make: webToken},
		{name: "the user and password of a web address", make: userAndPassword},
		{name: "the value of an Authorization header", make: authorizationValue},
		{name: "the value of a name that holds a word of a secret", make: namedValue},
		{name: "a long run of letters and digits", make: longRun},
		{name: "AIza", make: func(rt *rapid.T) credential {
			return tokenOf("AI" + "za" + word(rt, "body", alnum+"_-", 30, 30+boundary))
		}},
	}

	for _, name := range []string{
		"s" + "k", "p" + "k", "r" + "k", "gh" + "p", "gh" + "o", "gh" + "s", "gh" + "u", "gh" + "r", "github" + "_pat",
		"np" + "m", "gl" + "pat",
	} {
		for _, separator := range []string{"-", "_"} {
			prefix := name + separator

			specs = append(specs, credentialSpec{name: prefix, make: func(rt *rapid.T) credential {
				return tokenOf(prefix + word(rt, "body", alnum+"_-", 16, 16+boundary))
			}})
		}
	}

	for _, issuer := range []string{"AK" + "IA", "AS" + "IA"} {
		specs = append(specs, credentialSpec{name: issuer, make: func(rt *rapid.T) credential {
			return tokenOf(issuer + word(rt, "body", "0-9A-Z", 16, 16))
		}})
	}

	for _, kind := range []string{"a", "b", "p", "r", "s"} {
		prefix := "xo" + "x" + kind + "-"

		specs = append(specs, credentialSpec{name: prefix, make: func(rt *rapid.T) credential {
			return tokenOf(prefix + word(rt, "body", alnum+"-", 10, 10+boundary))
		}})
	}

	for _, scheme := range []string{"Bearer", "Basic"} {
		specs = append(specs, credentialSpec{name: scheme, make: func(rt *rapid.T) credential {
			spelled := rapid.SampledFrom([]string{scheme, strings.ToLower(scheme), strings.ToUpper(scheme)}).Draw(rt, "case")
			body := word(rt, "body", alnum+"._~+/=-", 12, 12+boundary)

			return credential{text: spelled + " " + body, secret: body}
		}})
	}

	return specs
}

// longRun is a run of 32 or more letters, digits, plus signs, underscores and equals signs that holds a letter and a
// digit, and is not a commit's 40 hexadecimal digits: letters that are no hexadecimal digits, so that it never is.
func longRun(rt *rapid.T) credential {
	secret := word(rt, "letters", "g-z", 16, 16+boundary) + word(rt, "digits", "0-9", 16, 16+boundary) +
		rapid.SampledFrom([]string{"", "_", "=", "+"}).Draw(rt, "last")

	return tokenOf(secret)
}

// namedValue is a value given to a name that holds a word of a secret.
func namedValue(rt *rapid.T) credential {
	name := secretName(rt)
	separator := rapid.SampledFrom([]string{"=", ": ", " is "}).Draw(rt, "separator")
	value := word(rt, "value", alnum+"!#%^*_+-", 6, 20)

	return credential{text: name + separator + value, secret: value, name: name}
}

// privateKeyBlock is a private key block.
func privateKeyBlock(rt *rapid.T) credential {
	body := word(rt, "body", "a-zA-Z0-9+/= \n", 20, 60)

	return credential{
		text:   "-----BEGIN RSA PRIVATE KEY-----\n" + body + "\n-----END RSA PRIVATE KEY-----",
		secret: strings.TrimSpace(body),
	}
}

// secretName draws a name that holds a word of a secret, with a word before it, after it, or neither, written in snake,
// kebab, camel or upper case.
func secretName(rt *rapid.T) string {
	parts := slices.Clone(rapid.SampledFrom([][]string{{"key"}, {"token"}, {"secret"}, {"password"}, {"passwd"}, {"pwd"},
		{"credential"}, {"private", "key"}}).Draw(rt, "word"))

	if rapid.Bool().Draw(rt, "has a word before") {
		before := rapid.SampledFrom([]string{"access", "signing", "client", "api", "db", "admin", "deploy", "github"})
		parts = slices.Insert(parts, 0, before.Draw(rt, "before"))
	}

	if rapid.Bool().Draw(rt, "has a word after") {
		parts = append(parts, rapid.SampledFrom([]string{"id", "value", "2", "prod"}).Draw(rt, "after"))
	}

	switch rapid.SampledFrom([]string{"snake", "kebab", "camel", "upper"}).Draw(rt, "case") {
	case "snake":
		return strings.Join(parts, "_")
	case "kebab":
		return strings.Join(parts, "-")
	case "upper":
		return strings.ToUpper(strings.Join(parts, "_"))
	default:
		var camel strings.Builder

		camel.WriteString(parts[0])

		for _, part := range parts[1:] {
			camel.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}

		return camel.String()
	}
}

// splitCredential generates a credential of a shape with a character that is not shown, or white space, put inside its
// secret: the secret as it was is in no text of the credential.
func splitCredential() *rapid.Generator[credential] {
	return rapid.Custom(func(rt *rapid.T) credential {
		found := credentialShape().Draw(rt, "credential")
		at := rapid.IntRange(1, max(len(found.secret)-1, 1)).Draw(rt, "split at")
		at = min(at, len(found.secret))
		inside := rapid.SampledFrom(notShown).Draw(rt, "inside")

		found.text = strings.Replace(found.text, found.secret, found.secret[:at]+inside+found.secret[at:], 1)

		return found
	})
}

// tokenOf is a credential that is all secret.
func tokenOf(secret string) credential {
	return credential{text: secret, secret: secret}
}

// userAndPassword is the user and password of a web address.
func userAndPassword(rt *rapid.T) credential {
	password := word(rt, "password", alnum+"!#%^*_+-", 6, 20)
	user := word(rt, "user", "a-z", 3, 8)

	return credential{text: "https://" + user + ":" + password + "@example.test/path", secret: password}
}

// webToken is a token of three parts joined by dots whose first part opens with eyJ, each part of 8 or more letters,
// digits, underscores and dashes.
func webToken(rt *rapid.T) credential {
	return tokenOf("ey" + "J" + word(rt, "head", alnum+"_-", 8, 8+boundary) + "." +
		word(rt, "body", alnum+"_-", 8, 8+boundary) + "." + word(rt, "tail", alnum+"_-", 8, 8+boundary))
}

// word draws a text of the characters of an alphabet, as long as a range says.
func word(rt *rapid.T, label, alphabet string, low, high int) string {
	return rapid.StringMatching(`[`+alphabet+`]{`+strconv.Itoa(low)+`,`+strconv.Itoa(high)+`}`).Draw(rt, label)
}
