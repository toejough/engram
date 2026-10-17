package auto

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Redact replaces each text of the text that is shaped as a credential with `[redacted]` and keeps the name of an
// assignment (793d4j): a private key block, a JSON web token, a key or token that opens with the name of its issuer, a
// Bearer or Basic credential of 12 characters or more, the user and password of a URL, the value of an Authorization
// header, the value given to a name that holds key, token, secret, password or the like, also when a sentence of up to
// six words stands between the name and the `is` or `:` that gives it, and a long run of letters and digits that is
// not a commit. A secret of another shape, such as a password that a sentence gives without `is` or `:`, or a token
// split over two lines, is not redacted.
func Redact(text string) string {
	text = wholeCredential.ReplaceAllString(text, redacted)
	text = urlUserinfo.ReplaceAllString(text, "${1}"+redacted+"@")
	text = authorization.ReplaceAllString(text, "${1}"+redacted)
	text = assignment.ReplaceAllString(text, "${1}"+redacted)

	return longToken.ReplaceAllStringFunc(text, func(run string) string {
		if holdsLetterAndDigit(run) && !commitID.MatchString(run) {
			return redacted
		}

		return run
	})
}

// unexported constants.
const (
	redacted = "[redacted]"
	// sensitiveName is a name that holds a word of a secret, in any case, with anything before and after the word; it
	// does not end with a full stop, which ends a sentence.
	sensitiveName = `(?:[\w.-]*(?:token|secret|passw(?:or)?d|pwd|credential|key|authorization)(?:[\w.-]*[\w-])?|` +
		`(?:[\w.-]*[_-])?pass)`
	// space is the white space of ECMAScript, and the mark that the window of the judgement writes a line break with:
	// the unicode spaces between a name and its value.
	space = `\s\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}\x{23ce}`
	// wordsBetween is how many words of a sentence may stand between a name and the `is` or `:` that gives its value.
	wordsBetween = 6
)

// unexported variables.
var (
	// assignment is a sensitive name given a value, by `=`, `:` or `is`, with up to wordsBetween words of the same
	// sentence between the name and the `is` or `:`: a word does not end a sentence, and holds no `=` or `:`.
	assignment = regexp.MustCompile(`(?i)(` + sensitiveName + `["']?(?:[` + space + `]+[^` + space + `=:]*[^` + space +
		`=:.;!?]){0,` + strconv.Itoa(wordsBetween) + `}?[` + space + `]*(?::|=|\bis\b)[` + space + `]*)` +
		`("[^"\n]*"|'[^'\n]*'|[^` + space + `]+)`)
	authorization = regexp.MustCompile(`(?i)(\bauthorization["']?[` + space + `]*[:=][` + space + `]*)[^\n\x{23ce}]*`)
	commitID      = regexp.MustCompile(`^[0-9a-f]{40}$`)
	longToken     = regexp.MustCompile(`[A-Za-z0-9+_=]{32,}`)
	urlUserinfo   = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^` + space + `:@/]+:[^` + space + `@/]+@`)
	// wholeCredential is any of the shapes that are replaced whole: a private key block, a JSON web token, the key or
	// token of an issuer, a Bearer or Basic credential.
	wholeCredential = regexp.MustCompile(
		`-----BEGIN [A-Z ]*PRIVATE KEY-----(?s:.*?)(?:-----END [A-Z ]*PRIVATE KEY-----|$)` +
			`|\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}` +
			`|\b(?:sk|pk|rk)[-_][A-Za-z0-9_-]{16,}` +
			`|\b(?:ghp|gho|ghs|ghu|ghr|github_pat|npm|glpat)[-_][A-Za-z0-9_-]{16,}` +
			`|\b(?:AKIA|ASIA)[0-9A-Z]{16}\b` +
			`|\bAIza[0-9A-Za-z_-]{30,}` +
			`|\bxox[abprs]-[A-Za-z0-9-]{10,}` +
			`|(?i:\bBearer[` + space + `]+[A-Za-z0-9._~+/=-]{12,})` +
			`|(?i:\bBasic[` + space + `]+[A-Za-z0-9._~+/=-]{12,})`)
)

// holdsLetterAndDigit reports whether the text holds both a letter and a digit.
func holdsLetterAndDigit(text string) bool {
	return strings.ContainsFunc(text, unicode.IsDigit) && strings.ContainsFunc(text, unicode.IsLetter)
}
