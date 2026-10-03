package chat

import (
	"reflect"
	"strings"
	"testing"
)

// An empty setting is what Claude Desktop sends when nobody typed anything into
// the field, which is the ordinary case. It is no domains and no error.
func TestAnEmptySettingExemptsNothing(t *testing.T) {
	for _, raw := range []string{"", " ", "\t", ",", " , , "} {
		got, err := Trusted(raw)
		if err != nil {
			t.Fatalf("Trusted(%q) refused the value: %v", raw, err)
		}
		if len(got) != 0 {
			t.Errorf("Trusted(%q) is %v, want no domains", raw, got)
		}
	}

}

// A listed domain exempts an address at that domain and at no other. A
// subdomain is not it, a domain that merely starts with it is not it, and a
// domain the listed one is the start of is not it: the person listed what they
// meant.
func TestAListedDomainExemptsAnEmailAtExactlyThatDomain(t *testing.T) {
	trusted, err := Trusted("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(trusted, []string{"example.com"}) {
		t.Fatalf("Trusted gave %v, want one domain example.com", trusted)
	}

	if got := Exempted("write to registry@example.com about it", trusted); !reflect.DeepEqual(got, []string{"registry@example.com"}) {
		t.Errorf("Exempted gave %v, want the address at the listed domain", got)
	}

	for _, text := range []string{
		"write to registry@sub.example.com about it",
		"write to registry@example.com.evil.example about it",
		"write to registry@xexample.com about it",
		"write to registry@example.company about it",
	} {
		if got := Exempted(text, trusted); got != nil {
			t.Errorf("Exempted(%q) gave %v, and that domain is not the listed one", text, got)
		}
	}
}

// What a person may type into the field: whole domains, separated by commas or
// spaces, in whatever case they used, and the same domain twice is once.
func TestTrustedTakesCommasAndSpacesAndLowercases(t *testing.T) {
	got, err := Trusted(" Example.COM, example.org  partner.example.net,example.com ")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"example.com", "example.org", "partner.example.net"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Trusted gave %v, want %v", got, want)
	}
}

// A value gdoc half understands is refused by name, with the setting named
// beside it, because the person reading the refusal is looking at that field.
// The four shapes are a wildcard, a web address, an address and a word with no
// dot.
func TestAMalformedDomainIsRefusedAndNamed(t *testing.T) {
	for _, raw := range []string{
		"*.example.com",
		"https://example.com",
		"registry@example.com",
		"example",
		"example.com, *.example.org",
		"example..com",
		"-example.com",
		"example.c0m",
	} {
		got, err := Trusted(raw)
		if err == nil {
			t.Errorf("Trusted(%q) gave %v and no error, want it refused", raw, got)
			continue
		}
		if got != nil {
			t.Errorf("Trusted(%q) refused and still gave %v, want no domains", raw, got)
		}
		if !strings.Contains(err.Error(), "Email domains that need no approval") {
			t.Errorf("the refusal of %q does not name the setting: %q", raw, err)
		}
	}

	// The bad value itself is in the sentence, so the person can see which of
	// the words they typed is the one gdoc could not read.
	_, err := Trusted("example.com, *.example.org")
	if err == nil {
		t.Fatal("a wildcard beside a good domain was accepted")
	}
	if !strings.Contains(err.Error(), "*.example.org") {
		t.Errorf("the refusal does not name the value that was wrong: %q", err)
	}
}

// Exempted is the fact a write answer states: which addresses in the text sat at
// a listed domain. The same address twice is one fact, whatever case it was
// written in, and no listed domain is no fact.
func TestExemptedNamesEachAddressAtAListedDomainOnce(t *testing.T) {
	trusted := []string{"example.com"}
	text := "ask registry@example.com, Registry@Example.com, audit@example.org or desk@example.com"

	got := Exempted(text, trusted)
	want := []string{"registry@example.com", "desk@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Exempted gave %v, want %v", got, want)
	}

	if got := Exempted(text, nil); got != nil {
		t.Errorf("Exempted with no listed domain gave %v, want nothing", got)
	}
}
