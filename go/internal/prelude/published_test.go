package prelude

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/guard"
)

// publishedDocument is markedDocument with the marker publish writes rather
// than the one a restyle writes.
func publishedDocument(start, end int) *docs.Document {
	d := markedDocument(start, end, false)
	d.Tabs[0].NamedRanges[0].Name = PublishedName
	return d
}

// A document publish made carries the house cover already. A run proposing a
// house cover would propose deleting it, contents list and all, so the run is
// refused and told to style without the cover.
func TestAPublishedCoverIsRefusedByARunThatProposesACover(t *testing.T) {
	// Arrange
	d := publishedDocument(1, 964)

	// Act
	_, err := Decide(d)

	// Assert
	if err == nil {
		t.Fatal("Decide() = nil, want a refusal: the document carries publish's own cover")
	}
	for _, want := range []string{PublishedName, "--fields"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// The two names are two markers. Published reads publish's and nothing else,
// and Markers reads the restyle's and nothing else.
func TestEachMarkerReaderReadsItsOwnName(t *testing.T) {
	// Arrange
	published := publishedDocument(1, 964)
	proposed := markedDocument(1, 964, false)

	// Act
	fromPublished, err := Published(published)
	if err != nil {
		t.Fatalf("Published() = %v", err)
	}
	proposedAsPublished, err := Published(proposed)
	if err != nil {
		t.Fatalf("Published() = %v", err)
	}
	publishedAsProposed, err := Markers(published)
	if err != nil {
		t.Fatalf("Markers() = %v", err)
	}

	// Assert
	if len(fromPublished) != 1 || fromPublished[0].Start != 1 || fromPublished[0].End != 964 {
		t.Errorf("Published() = %+v, want the one marker over [1,964)", fromPublished)
	}
	if len(proposedAsPublished) != 0 {
		t.Errorf("Published() read a restyle's marker: %+v", proposedAsPublished)
	}
	if len(publishedAsProposed) != 0 {
		t.Errorf("Markers() read publish's marker: %+v", publishedAsProposed)
	}
}

// A published marker is held to the same shape as a restyle's: one span of one
// tab's body. One broken across the author's words is refused, because a
// restyle walking past it would leave their words unstyled.
func TestAPublishedMarkerBrokenAcrossTheAuthorsTextIsRefused(t *testing.T) {
	// Arrange
	d := publishedDocument(1, 964)
	d.Tabs[0].NamedRanges[0].Ranges = append(d.Tabs[0].NamedRanges[0].Ranges,
		d.Tabs[0].NamedRanges[0].Ranges[0])
	d.Tabs[0].NamedRanges[0].Ranges[1].Start = 970
	d.Tabs[0].NamedRanges[0].Ranges[1].End = 980

	// Act
	_, err := Published(d)

	// Assert
	if err == nil {
		t.Fatal("Published() = nil, want a refusal for a marker with the author's text inside it")
	}
}

// publish writes its marker on a document the guard's own create made, which is
// the full level, so it needs no grant. The pin is the real policy judging the
// real bytes, the way TestTheMarkerRequestIsWhatTheGuardGrants is for the
// restyle's marker.
func TestThePublishedMarkerNeedsNoGrantOnACreatedDocument(t *testing.T) {
	// Arrange
	p := guard.NewPolicy()
	p.Learn("DOC1")
	body, err := json.Marshal(map[string]any{"requests": []map[string]any{PublishedRequest(1, 964)}})
	if err != nil {
		t.Fatal(err)
	}
	at, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatal(err)
	}

	// Act, Assert
	if err := p.Judge("POST", at, body); err != nil {
		t.Fatalf("the guard refused publish's marker on a document it created: %v", err)
	}
}

// And a document that was handed in, rather than created, cannot take it. A
// restyle at the in-place level is granted one marker, by name, and this is not
// that name.
func TestThePublishedMarkerIsRefusedOnAHandedInDocument(t *testing.T) {
	// Arrange
	p := guard.NewPolicy()
	p.AllowFile("DOC1", guard.LevelSuggest)
	p.GrantInPlace("DOC1")
	p.AllowMarker(MarkerName, 1, 964)
	body, err := json.Marshal(map[string]any{"requests": []map[string]any{PublishedRequest(1, 964)}})
	if err != nil {
		t.Fatal(err)
	}
	at, err := url.Parse("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if err != nil {
		t.Fatal(err)
	}

	// Act, Assert
	if err := p.Judge("POST", at, body); err == nil {
		t.Fatal("the guard carried publish's marker on a document granted only a restyle's")
	}
}
