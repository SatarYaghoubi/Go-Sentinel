package scanner

import "regexp"

// Severity describes how dangerous a matched rule is.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Rule is a single signature used to inspect file content.
type Rule struct {
	ID          string
	Name        string
	Description string
	Severity    Severity
	pattern     *regexp.Regexp
}

// Match reports whether the rule's pattern is present in the given data.
func (r Rule) Match(data []byte) bool {
	return r.pattern.Match(data)
}

// defaultRules is a small, illustrative signature set. In a real system these
// would be loaded from a regularly-updated feed; here they are embedded so the
// service runs with zero configuration.
var defaultRules = []Rule{
	{
		ID:          "EICAR-001",
		Name:        "EICAR test signature",
		Description: "Standard antivirus test string.",
		Severity:    SeverityCritical,
		pattern:     regexp.MustCompile(`EICAR-STANDARD-ANTIVIRUS-TEST-FILE`),
	},
	{
		ID:          "SHELL-001",
		Name:        "Embedded reverse shell",
		Description: "Common reverse-shell invocation pattern.",
		Severity:    SeverityHigh,
		pattern:     regexp.MustCompile(`(?i)/bin/(ba)?sh\s+-i\s+>&\s*/dev/tcp/`),
	},
	{
		ID:          "PHP-001",
		Name:        "PHP code execution",
		Description: "Use of dynamic execution functions in uploaded content.",
		Severity:    SeverityHigh,
		pattern:     regexp.MustCompile(`(?i)\b(eval|system|exec|passthru|shell_exec)\s*\(`),
	},
	{
		ID:          "SECRET-001",
		Name:        "Hardcoded private key",
		Description: "PEM-encoded private key material embedded in upload.",
		Severity:    SeverityMedium,
		pattern:     regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`),
	},
	{
		ID:          "MACRO-001",
		Name:        "Office auto-exec macro",
		Description: "Auto-executing macro entry point in document.",
		Severity:    SeverityMedium,
		pattern:     regexp.MustCompile(`(?i)\b(AutoOpen|Workbook_Open|Document_Open)\b`),
	},
}

// DefaultRules returns a copy of the embedded rule set.
func DefaultRules() []Rule {
	rules := make([]Rule, len(defaultRules))
	copy(rules, defaultRules)
	return rules
}
