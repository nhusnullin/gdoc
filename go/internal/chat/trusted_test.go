package chat

import (
	"reflect"
	"strings"
	"testing"
)

// An empty setting is what Claude Desktop sends when nobody typed anything into
// the field, which is the ordinary case. It is no domains and no error, and the
// Link rule asks about every address as it would with no setting at all.
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

	none, err := Trusted("")
	if err != nil {
		t.Fatal(err)
	}
	h, err := Rules(plainReply("write to registry@example.com about it"), settled(), none, ruleNow)
	if err != nil {
		t.Fatalf("the rules refused the write outright: %v", err)
	}
	if h == nil || h.Rule != "Link" {
		t.Errorf("an address was not held for Link with an empty setting, and %v is what came back", h)
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

	exempt, err := Rules(plainReply("write to registry@example.com about it"), settled(), trusted, ruleNow)
	if err != nil {
		t.Fatalf("the rules refused the write outright: %v", err)
	}
	if exempt != nil {
		t.Errorf("an address at the listed domain was held for %q", exempt.Rule)
	}

	for _, text := range []string{
		"write to registry@sub.example.com about it",
		"write to registry@example.com.evil.example about it",
		"write to registry@xexample.com about it",
		"write to registry@example.company about it",
	} {
		held, err := Rules(plainReply(text), settled(), trusted, ruleNow)
		if err != nil {
			t.Fatalf("the rules refused %q outright: %v", text, err)
		}
		if held == nil || held.Rule != "Link" {
			t.Errorf("%q was not held for Link, and that domain is not the listed one", text)
		}
	}
}

// A link is never exempt, whatever its domain. The setting is about an address a
// colleague writes in a reply, and a link is where a reader is sent.
func TestALinkAtAListedDomainIsStillHeld(t *testing.T) {
	trusted, err := Trusted("example.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"the fees are at https://example.com/fees",
		"the fees are at www.example.com",
		"the fees are at example.com/fees",
	} {
		held, err := Rules(plainReply(text), settled(), trusted, ruleNow)
		if err != nil {
			t.Fatalf("the rules refused %q outright: %v", text, err)
		}
		if held == nil || held.Rule != "Link" {
			t.Errorf("%q was not held for Link, and a link is never exempt", text)
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
