package update

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// recorded is the releases listing as api.github.com answered it, kept in
// testdata so the choice is made over what GitHub actually sends rather than
// over a shape written from memory.
func recorded(t *testing.T) []Entry {
	t.Helper()
	raw, err := os.ReadFile("testdata/releases.json")
	if err != nil {
		t.Fatal(err)
	}
	var entries []Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("the recorded listing does not decode: %v", err)
	}
	return entries
}

func TestChoosePicksTheHighestOfTheChannelForThePlatform(t *testing.T) {
	cases := []struct {
		name     string
		channel  Channel
		platform string
		want     string
		asset    string
	}{
		{"stable on this machine", Stable, "darwin-arm64", "v2.1.0", "gdoc-v2.1.0-darwin-arm64.zip"},
		{"stable on the other mac", Stable, "darwin-amd64", "v2.1.0", "gdoc-v2.1.0-darwin-amd64.zip"},
		{"nightly on this machine", Nightly, "darwin-arm64", "v2.1.2", "gdoc-v2.1.2-darwin-arm64.zip"},
		{"nightly on the other mac", Nightly, "darwin-amd64", "v2.1.2", "gdoc-v2.1.2-darwin-amd64.zip"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Choose(recorded(t), c.channel, c.platform)
			if err != nil {
				t.Fatalf("Choose: %v", err)
			}
			if got.Version.String() != c.want {
				t.Errorf("chose %s, want %s", got.Version, c.want)
			}
			if got.Tag != c.want {
				t.Errorf("tag = %q, want %q", got.Tag, c.want)
			}
			if got.AssetName != c.asset {
				t.Errorf("asset = %q, want %q", got.AssetName, c.asset)
			}
			wantURL := "https://github.com/nhusnullin/gdoc/releases/download/" + c.want + "/" + c.asset
			if got.AssetURL != wantURL {
				t.Errorf("asset URL = %q, want %q", got.AssetURL, wantURL)
			}
			wantSums := "https://github.com/nhusnullin/gdoc/releases/download/" + c.want + "/SHA256SUMS-" + c.want
			if got.ChecksumsURL != wantSums {
				t.Errorf("checksums URL = %q, want %q", got.ChecksumsURL, wantSums)
			}
		})
	}
}

func TestChooseRefusesWhenThePlatformHasNoAssetInTheLatestRelease(t *testing.T) {
	// v2.0.0 carried a Windows zip and v2.1.0 does not. The answer is a
	// refusal naming the platform, never the older release: an update that
	// silently installs something other than the latest is worse than one
	// that says it cannot.
	_, err := Choose(recorded(t), Stable, "windows-amd64")
	if err == nil {
		t.Fatal("Choose found a Windows release in a listing whose latest stable has none")
	}
	if !strings.Contains(err.Error(), "windows-amd64") || !strings.Contains(err.Error(), "v2.1.0") {
		t.Errorf("the refusal says %q, want it to name the platform and the release", err)
	}
}

func TestChooseSkipsADraftAndATagThatIsNotAVersion(t *testing.T) {
	// The listing holds a draft v2.2.0 and a moving tag called "nightly".
	// A draft's assets are not public and an unparseable tag cannot be
	// compared, so neither is ever chosen.
	got, err := Choose(recorded(t), Nightly, "darwin-arm64")
	if err != nil {
		t.Fatalf("Choose: %v", err)
	}
	if got.Version.String() != "v2.1.2" {
		t.Errorf("chose %s, want v2.1.2, which is the highest release that is neither a draft nor an unreadable tag", got.Version)
	}
}

func TestChooseRefusesAnEmptyListing(t *testing.T) {
	_, err := Choose(nil, Stable, "darwin-arm64")
	if err == nil {
		t.Fatal("Choose found a release in an empty listing")
	}
	if !strings.Contains(err.Error(), "no release") {
		t.Errorf("the refusal says %q, want it to say no release was found", err)
	}
}

func TestChooseRefusesAChecksumFileThatIsMissing(t *testing.T) {
	entries := []Entry{{
		Tag:    "v2.1.0",
		Assets: []Asset{{Name: "gdoc-v2.1.0-darwin-arm64.zip", URL: "https://github.com/nhusnullin/gdoc/releases/download/v2.1.0/gdoc-v2.1.0-darwin-arm64.zip"}},
	}}
	_, err := Choose(entries, Stable, "darwin-arm64")
	if err == nil {
		t.Fatal("Choose took a release with no checksum file; nothing could then be verified")
	}
	if !strings.Contains(err.Error(), "SHA256SUMS-v2.1.0") {
		t.Errorf("the refusal says %q, want it to name the checksum file", err)
	}
}

func TestAnAssetNameIsBuiltFromTheTagAndThePlatform(t *testing.T) {
	if got := AssetName("v2.1.0", "darwin-arm64"); got != "gdoc-v2.1.0-darwin-arm64.zip" {
		t.Errorf("AssetName = %q, want gdoc-v2.1.0-darwin-arm64.zip", got)
	}
	if got := ChecksumsName("v2.1.0"); got != "SHA256SUMS-v2.1.0" {
		t.Errorf("ChecksumsName = %q, want SHA256SUMS-v2.1.0", got)
	}
}

func TestThePlatformIsTheGoPairTheBuildNames(t *testing.T) {
	// The zips are named after GOOS and GOARCH, which is what `make dist`
	// builds them from, so the running binary can name its own.
	if got := Platform("darwin", "arm64"); got != "darwin-arm64" {
		t.Errorf("Platform = %q, want darwin-arm64", got)
	}
	if got := Platform("windows", "amd64"); got != "windows-amd64" {
		t.Errorf("Platform = %q, want windows-amd64", got)
	}
}
