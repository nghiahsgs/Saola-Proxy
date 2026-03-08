package scanner

import "regexp"

// Compiled regexes at package level for performance.
var (
	reAWSAccessKey    = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	reGitHubToken     = regexp.MustCompile(`(ghp|gho|ghs|ghr|github_pat)_[A-Za-z0-9_]{36,}`)
	reStripeKey       = regexp.MustCompile(`(sk|pk)_(test|live)_[A-Za-z0-9]{24,}`)
	reGenericAPIKey   = regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|api_secret|access_token)\s*[:=]\s*['"]?([A-Za-z0-9_\-]{20,})`)
	rePrivateKey      = regexp.MustCompile(`-----BEGIN\s+(?:RSA|EC|DSA|OPENSSH|PGP)\s+PRIVATE\s+KEY-----`)
	reJWT             = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)
	reConnectionStr   = regexp.MustCompile(`(?i)(?:postgres|mysql|mongodb|redis|amqp)://[^\s'"]+`)
	reEmail           = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	reSSN             = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	reCreditCard      = regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`)
	rePhoneUS         = regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`)
	reIPv4            = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)
	reEnvVariable     = regexp.MustCompile(`(?i)(?:PASSWORD|SECRET|TOKEN|PRIVATE_KEY|CREDENTIALS)\s*[:=]\s*['"]?([^\s'"]{8,})`)
)

// RegisterBuiltins registers all 13 built-in PII/secret detection patterns.
func RegisterBuiltins(r *PatternRegistry) {
	r.Register(Pattern{
		Name:        "aws-access-key",
		Category:    "credential",
		Regex:       reAWSAccessKey,
		Description: "AWS Access Key ID",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "github-token",
		Category:    "secret",
		Regex:       reGitHubToken,
		Description: "GitHub personal access token or OAuth token",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "stripe-key",
		Category:    "secret",
		Regex:       reStripeKey,
		Description: "Stripe API key (live or test)",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "generic-api-key",
		Category:    "secret",
		Regex:       reGenericAPIKey,
		Description: "Generic API key or access token assignment",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "private-key",
		Category:    "credential",
		Regex:       rePrivateKey,
		Description: "PEM-encoded private key header",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "jwt",
		Category:    "secret",
		Regex:       reJWT,
		Description: "JSON Web Token",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "connection-string",
		Category:    "credential",
		Regex:       reConnectionStr,
		Description: "Database or message broker connection string",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "email",
		Category:    "pii",
		Regex:       reEmail,
		Description: "Email address",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "ssn",
		Category:    "pii",
		Regex:       reSSN,
		Description: "US Social Security Number",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "credit-card",
		Category:    "pii",
		Regex:       reCreditCard,
		Description: "Credit card number (Visa, MasterCard, Amex, Discover)",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "phone-us",
		Category:    "pii",
		Regex:       rePhoneUS,
		Description: "US phone number",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "ipv4-address",
		Category:    "pii",
		Regex:       reIPv4,
		Description: "IPv4 address",
		Enabled:     true,
	})
	r.Register(Pattern{
		Name:        "env-variable",
		Category:    "secret",
		Regex:       reEnvVariable,
		Description: "Environment variable containing a sensitive value",
		Enabled:     true,
	})
}
