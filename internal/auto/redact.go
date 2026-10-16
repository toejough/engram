package auto

import (
	"regexp"
	"strconv"
	"unicode/utf8"
)

// unexported constants.
const (
	// the shortest run of token-like characters at the edge of a cut text that is hidden
	edgeRunMin = "16"
	hexBits    = 8
	hexRadix   = 16
	keepName   = "${1}" + redacted
	redacted   = "[redacted]"
	runChars   = `[A-Za-z0-9_+/=-]`
	// a name that says its value is a secret, and the value after a name: quoted (with spaces) or the next run
	sensitive = `(?:[\w.-]*(?:token|secret|passw(?:or)?d|pwd|api[-_ ]?key|apikey|credential|private[-_ ]?key|` +
		`authorization|[_-]key)[\w.-]*|(?:[\w.-]*[_-])?pass)`
	// the white space of ECMAScript, as the body of a character class
	space = `\s\v\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`
	value = `("[^"\n]*"|'[^'\n]*'|[^` + space + `]+)`
)

// unexported variables.
var (
	// an Authorization header: the whole line is the value
	authorization = regexp.MustCompile(`(?i)(\bauthorization["']?[` + space + `]*[:=][` + space + `]*)[^\n]*`)
	awsKey        = regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)
	basic         = regexp.MustCompile(`(?i)\bBasic[` + space + `]+[A-Za-z0-9._~+/=-]{12,}`)
	bearer        = regexp.MustCompile(`(?i)\bBearer[` + space + `]+[A-Za-z0-9._~+/=-]+`)
	digit, letter = regexp.MustCompile(`\d`), regexp.MustCompile(`[A-Za-z]`)
	gitSHA        = regexp.MustCompile(`^[0-9a-f]{40}$`)
	googleKey     = regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{30,}`)
	headRun       = regexp.MustCompile(`^` + runChars + `{` + edgeRunMin + `,}`)
	hostedToken   = regexp.MustCompile(`\b(?:ghp|gho|ghs|ghu|ghr|github_pat|npm|glpat)[-_][A-Za-z0-9_-]{16,}`)
	// a value assigned to a name that ends in key (`KEY=`, `monkey=`), and to any name that says it is a secret
	keyAssigned = regexp.MustCompile(`(?i)([\w.-]*key["']?[` + space + `]*=[` + space + `]*)` + value)
	longToken   = regexp.MustCompile(`[A-Za-z0-9+_=]{32,}`)
	// a percent sign that is not followed by two hexadecimal digits
	malformedPercent = regexp.MustCompile(`%(?:[^0-9A-Fa-f]|[0-9A-Fa-f][^0-9A-Fa-f]|[0-9A-Fa-f]?$)`)
	named            = regexp.MustCompile(
		`(?i)(` + sensitive + `["']?[` + space + `]*(?::|=|\bis\b)[` + space + `]*)` + value)
	percentEscape = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)
	percentToken  = regexp.MustCompile(`[^` + space + `]*%[0-9A-Fa-f]{2}[^` + space + `]*`)
	privateKey    = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----(?s:.*?)(?:-----END [A-Z ]*PRIVATE KEY-----|$)`)
	providerKey   = regexp.MustCompile(`\b(?:sk|pk|rk)[-_][A-Za-z0-9_-]{16,}`)
	slackToken    = regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`)
	// a run of token-like characters at an edge of a text that was cut: what is left of a credential the cut split, which
	// no pattern recognizes in a fragment (793b471)
	tailRun      = regexp.MustCompile(runChars + `{` + edgeRunMin + `,}$`)
	tokenFollows = regexp.MustCompile(`^` + runChars + `{4}`)
	urlUserinfo  = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^` + space + `:@/]+:[^` + space + `@/]+@`)
	webToken     = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`)
	wrappedLines = regexp.MustCompile(`(?m)(?:^[ \t]*[A-Za-z0-9+/=_-]{20,}[ \t]*\r?\n){2,}`)
	wrappedToken = regexp.MustCompile(`(` + runChars + `{12,})[ \t]*\r?\n[ \t]*`)
)

// cutEnd is text.slice(0, limit), in characters.
func cutEnd(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit])
	}

	return text
}

// cutStart is text.slice(-limit), in characters.
func cutStart(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[len(runes)-limit:])
	}

	return text
}

// decode is JavaScript's decodeURIComponent; a text that is not a well-formed encoding stays as it was.
func decode(text string) string {
	if malformedPercent.MatchString(text) {
		return text
	}

	decoded := percentEscape.ReplaceAllStringFunc(text, func(escape string) string {
		number, _ := strconv.ParseUint(escape[1:], hexRadix, hexBits)

		return string([]byte{byte(number)})
	})

	return map[bool]string{true: decoded, false: text}[utf8.ValidString(decoded)]
}

// maskHead hides the run of token-like characters that starts the text, the end of a credential a cut split.
func maskHead(text string) string { return headRun.ReplaceAllLiteralString(text, redacted) }

// maskTail hides the run of token-like characters that ends the text, the start of a credential a cut split.
func maskTail(text string) string { return tailRun.ReplaceAllLiteralString(text, redacted) }

// redact hides the credentials a text holds.
func redact(text string) string {
	// a token wrapped over lines is judged joined; the joined text is kept only when it held a secret
	joined, last := "", 0

	for _, span := range wrappedToken.FindAllStringSubmatchIndex(text, -1) {
		if tokenFollows.MatchString(text[span[1]:]) {
			joined, last = joined+text[last:span[0]]+text[span[2]:span[3]], span[1]
		}
	}

	if joined += text[last:]; redactOnce(joined) == joined {
		joined = text
	}

	return percentToken.ReplaceAllStringFunc(redactOnce(joined), func(found string) string {
		if decoded := decode(found); redactOnce(decoded) != decoded {
			return redacted
		}

		return found
	})
}

// redactOnce is one pass of every pattern. A 40-character hex run is a git commit hash far more often than a secret.
func redactOnce(text string) string {
	whole := []*regexp.Regexp{privateKey, webToken, providerKey, hostedToken, awsKey, googleKey, slackToken, bearer, basic}

	for _, pattern := range whole {
		text = pattern.ReplaceAllString(text, redacted)
	}

	text = urlUserinfo.ReplaceAllString(text, "${1}"+redacted+"@")

	for _, pattern := range []*regexp.Regexp{authorization, keyAssigned, named} {
		text = pattern.ReplaceAllString(text, keepName)
	}

	mixed := func(run string) bool { return digit.MatchString(run) && letter.MatchString(run) }
	text = wrappedLines.ReplaceAllStringFunc(text, func(run string) string {
		if mixed(run) {
			return redacted + "\n"
		}

		return run
	})

	return longToken.ReplaceAllStringFunc(text, func(run string) string {
		if mixed(run) && !gitSHA.MatchString(run) {
			return redacted
		}

		return run
	})
}
