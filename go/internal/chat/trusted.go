// The one setting a person of their own accord may type into Claude Desktop:
// the email domains an address may sit at without a card.

package chat

import (
	"fmt"
	"strings"
)

// TrustedSetting is the field's own name, as the person reads it in Claude
// Desktop. It is in every sentence a malformed value gets, because the person
// fixing it is looking at that field and nothing else says which one it is. The
// extension's manifest shows the field under the same words.
const TrustedSetting = `the gdoc extension setting "Email domains that need no approval"`

// trustedSeparators are what a person puts between two domains. Both are taken,
// because a field with no example in it gets a comma from one person and a space
// from the next.
const trustedSeparators = ", \t\n\r"

// Trusted reads the setting into the list the Link rule asks.
//
// Whole lowercased domains only. An empty value is no domains and no error,
// which is what Claude Desktop sends when nobody typed anything in the field
// (docs/v2/MEASURED.md, measurement 1): TestAnEmptySettingExemptsNothing. A
// value gdoc half understands is refused by name rather than read as best it
// can, because a domain read wrongly is a card that never appears:
// TestAMalformedDomainIsRefusedAndNamed.
//
// Nothing in gdoc suggests filling this field in. It is decision 17 of the
// milestone 14 specification: the person went looking for it, and what the
// binary says about it is a fact and never an offer.
func Trusted(raw string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, field := range strings.FieldsFunc(raw, func(r rune) bool {
		return strings.ContainsRune(trustedSeparators, r)
	}) {
		domain := strings.ToLower(field)
		if why := whyNotADomain(domain); why != "" {
			return nil, fmt.Errorf("%s holds %q, which is not an email domain: %s. Whole domains only, like example.com, separated by commas",
				TrustedSetting, field, why)
		}
		if seen[domain] {
			continue
		}
		seen[domain] = true
		out = append(out, domain)
	}
	return out, nil
}

// whyNotADomain is the one short reason a value is no domain, or the empty
// string where it is one. The four shapes a person actually types are named
// first, so the sentence says what they wrote rather than what a domain is.
func whyNotADomain(domain string) string {
	switch {
	case strings.Contains(domain, "*"):
		return "it is a wildcard, and gdoc matches a whole domain and never a pattern"
	case strings.Contains(domain, "://") || strings.Contains(domain, ":"):
		return "it is a web address, and this field takes the part after the @ of an email address"
	case strings.Contains(domain, "@"):
		return "it is an email address, and this field takes the domain after the @ alone"
	case strings.Contains(domain, "/"):
		return "it carries a path, and a domain carries none"
	case !strings.Contains(domain, "."):
		return "it is one word with no dot, and every email domain has at least one"
	}
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if label == "" {
			return "a part of it between two dots is empty"
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Sprintf("the part %q starts or ends with a hyphen", label)
		}
		for _, r := range label {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
				continue
			}
			return fmt.Sprintf("the part %q holds %q, and a domain holds letters, digits and hyphens", label, r)
		}
	}
	last := labels[len(labels)-1]
	if len(last) < 2 {
		return fmt.Sprintf("it ends in %q, which is too short to be the last part of a domain", last)
	}
	for _, r := range last {
		if r < 'a' || r > 'z' {
			return fmt.Sprintf("it ends in %q, and the last part of a domain is letters", last)
		}
	}
	return ""
}

// Exempted names every email address in a text sitting at exactly one of the
// listed domains, each once.
//
// It is the fact a write answer states: a check the person loosened is a check
// the answer says out loud, so the one written-down loosening in gdoc is one
// nobody has to go looking for. The addresses come back as they were written,
// because that is what the document will carry:
// TestExemptedNamesEachAddressAtAListedDomainOnce.
//
// It is the same reading the Link rule does, through the same isTrusted, so the
// fact and the rule cannot say different things about one address.
func Exempted(text string, trusted []string) []string {
	if len(trusted) == 0 {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, address := range emailPattern.FindAllString(text, -1) {
		if !isTrusted(address, trusted) {
			continue
		}
		key := strings.ToLower(address)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, address)
	}
	return out
}
