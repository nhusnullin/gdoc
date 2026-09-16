package update

import (
	"fmt"
	"sort"
)

// Channel is which of the two kinds of release a run is looking for. The
// channel is read out of the tag and never out of GitHub's own prerelease
// flag: a nightly is an ordinary release of this repository, and what makes it
// a nightly is that its patch number is not zero.
type Channel int

const (
	// Stable is a release cut by hand, x.y.0.
	Stable Channel = iota
	// Nightly is the highest release of either kind. A person who asked for
	// the nightly channel asked for the newest thing there is, and on the day
	// after a stable tag that is the stable tag.
	Nightly
)

func (c Channel) String() string {
	if c == Nightly {
		return "nightly"
	}
	return "stable"
}

// Asset is one file published with a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Entry is one release as api.github.com lists it. Only the four fields gdoc
// reads are named; the rest of the answer is discarded by encoding/json.
type Entry struct {
	Tag    string  `json:"tag_name"`
	Draft  bool    `json:"draft"`
	Assets []Asset `json:"assets"`
}

// Release is a release gdoc could install on one platform: the version, and
// the two URLs the install needs.
type Release struct {
	Version      Version
	Tag          string
	AssetName    string
	AssetURL     string
	ChecksumsURL string
}

// AssetName is what the release workflow names a platform's zip.
func AssetName(tag, platform string) string { return "gdoc-" + tag + "-" + platform + ".zip" }

// ChecksumsName is what the release workflow names the checksum file. There is
// one per release, covering every zip in it.
func ChecksumsName(tag string) string { return "SHA256SUMS-" + tag }

// Platform is the pair `make dist` builds a binary for, and the pair a running
// binary names itself with through runtime.GOOS and runtime.GOARCH.
func Platform(goos, goarch string) string { return goos + "-" + goarch }

// Choose picks the release a run would install, out of the listing, for one
// channel and one platform.
//
// A draft is skipped: its assets are not public, so choosing one would be a
// download that 404s. A tag nothing can parse is skipped, because a moving tag
// cannot be compared with the version in somebody's binary.
//
// The highest release of the channel is the only candidate. A latest release
// that carries no zip for this platform is a refusal naming the platform, and
// never a quiet fall back to the release before it: an update that installs
// something other than the newest is worse than one that says what is missing.
func Choose(entries []Entry, channel Channel, platform string) (Release, error) {
	candidates := make([]Entry, 0, len(entries))
	versions := make(map[string]Version, len(entries))
	for _, e := range entries {
		if e.Draft {
			continue
		}
		v, err := Parse(e.Tag)
		if err != nil {
			continue
		}
		if channel == Stable && !v.IsStable() {
			continue
		}
		candidates = append(candidates, e)
		versions[e.Tag] = v
	}
	if len(candidates) == 0 {
		return Release{}, fmt.Errorf("the listing holds no release on the %s channel", channel)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return Compare(versions[candidates[i].Tag], versions[candidates[j].Tag]) > 0
	})
	latest := candidates[0]
	return release(latest, versions[latest.Tag], platform)
}

// release turns the chosen entry into the two URLs, or says which file the
// release does not carry.
func release(e Entry, v Version, platform string) (Release, error) {
	zip := AssetName(e.Tag, platform)
	sums := ChecksumsName(e.Tag)
	found := make(map[string]string, len(e.Assets))
	for _, a := range e.Assets {
		found[a.Name] = a.URL
	}
	if found[zip] == "" {
		return Release{}, fmt.Errorf("release %s carries no %s, so there is nothing to install on %s", e.Tag, zip, platform)
	}
	if found[sums] == "" {
		return Release{}, fmt.Errorf("release %s carries no %s, so nothing it holds could be verified", e.Tag, sums)
	}
	return Release{
		Version:      v,
		Tag:          e.Tag,
		AssetName:    zip,
		AssetURL:     found[zip],
		ChecksumsURL: found[sums],
	}, nil
}
