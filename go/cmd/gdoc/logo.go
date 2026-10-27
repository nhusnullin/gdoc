// Where the mark in the first page's header comes from.
//
// A house style names the logo in one of two places. `logo.base64` is bytes in
// the file, and nothing is fetched. `logo.drive` is one file in the firm's
// Drive, and then the bytes come through internal/asset: one request a day,
// cached beside the token, and never a fall back to an unbranded document.
//
// `build` and `publish` are the only two commands that render a note, so they
// are the only two that ask. `export`, `restyle` and `help marks` read the
// house style for other reasons and never come through here, which is what
// keeps every read command as offline as it was.

package main

import (
	"context"
	"fmt"
	"time"

	"gdoc/internal/asset"
	"gdoc/internal/guard"
	"gdoc/internal/house"
)

// logoFromFile is what logo.source says when the bytes were in the house
// style itself. No request was made, there is nothing to age, and the other
// fields are left out.
const logoFromFile = "file"

// logoCeiling is the whole of what the logo may cost a render. The guard
// bounds one request at five minutes, which is there so a hung endpoint does
// not hang the CLI forever; it is far too long for this one. A build is a
// local thing a person waits on, the cache answers every run but the first of
// the day, and a network that hangs rather than refusing would otherwise sit
// on that build for five minutes, doubled by the retry after a 401, before
// using the mark it already had.
//
// Twenty seconds rather than the two the update check gets, because that one
// is a fetch nobody asked for and this one is a document that cannot be built
// without it. The first build on a machine has no cache behind it, and the
// deadline covers a token refresh as well as the read.
//
// It is a variable so a test can shorten it, as accountCeiling is.
// TestTheLogoCeilingIsTwentySeconds states the literal and
// TestAHungDriveCostsTheCeilingAndThenTheCacheAnswers holds the fallback.
var logoCeiling = 20 * time.Second

// logoData is where this run's mark came from. Every field is a fact: nothing
// here says the document is branded correctly, only which bytes went into it,
// how old they are, and what stopped Drive from being asked or from
// answering.
//
// The age is in seconds rather than as a duration somebody has to parse,
// because the object is read by a skill and a bare number with its unit in
// the key is the one shape that cannot be misread.
type logoData struct {
	Source     string `json:"source"`
	ID         string `json:"id,omitempty"`
	FetchedAt  string `json:"fetched_at,omitempty"`
	AgeSeconds int64  `json:"age_seconds,omitempty"`
	Reason     string `json:"reason,omitempty"`
	CacheError string `json:"cache_error,omitempty"`
}

// logoReach is one run's door to the house logo: a policy carrying the asset
// grant alone, and a session opened at the moment the room asks for the bytes
// rather than before.
//
// Opening late is the whole of why it is a type. A build whose cache is fresh
// asks Drive nothing, and a session opened in front of that question would
// load the token, refresh it when it had expired and write it back, for a
// request that was never going to be made. A machine with no sign-in gets the
// login named as the reason the cache was used, which is what internal/asset
// reports it as, rather than as a refusal of the whole run.
type logoReach struct {
	policy *guard.Policy
	// opened is what the session said about itself, kept because a refresh or
	// a guard refusal is the caller's to report and this reach is handed to a
	// room that prints nothing.
	opened []string
}

// FileBytes is asset.Drive: the one read this reach makes.
func (r *logoReach) FileBytes(ctx context.Context, fileID string, limit int64) ([]byte, error) {
	s, err := openSession(r.policy)
	if err != nil {
		return nil, err
	}
	raw, err := s.FileBytes(ctx, fileID, limit)
	r.opened = append(r.opened, s.Warnings()...)
	return raw, err
}

// logoFor is the mark this run renders with, the facts about it, and what is
// worth saying out loud about where it came from.
//
// A house style carrying its own bytes is the whole of the function. A house
// style naming a file in Drive goes through internal/asset, which asks Drive
// at most once a day, holds the answer to the mime, byte ceiling and side
// ceiling, and refuses rather than returning bytes it did not check.
//
// The two things a person hears about are the two worth hearing: a mark Drive
// was asked for and did not give, so the document carries the cached copy
// instead, and a mark that was fetched and could not be cached. A cache used
// because Drive was already asked today is the designed path and earns no
// line; the envelope still says the bytes came from it. Neither case stops
// the run, and nothing here judges either.
// TestBuildReportsTheLogoSourceFile, TestBuildReportsTheLogoSourceDrive,
// TestBuildReportsTheLogoSourceCache and
// TestBuildWithNoTokenAndNoCacheRefusesNamingTheLogin. A wrong Drive answer
// refuses even with a good cache: TestBuildRefusesAWrongFileInDriveWithAGoodCache.
func logoFor(ctx context.Context, cfg *house.Config) ([]byte, logoData, []string, error) {
	id := cfg.LogoDrive()
	if id == "" {
		raw, err := cfg.LogoBytes()
		if err != nil {
			return nil, logoData{}, nil, err
		}
		return raw, logoData{Source: logoFromFile}, nil, nil
	}

	// One grant, one file, for this call. The want is the house file's own
	// statement about the picture's mime.
	p := guard.NewPolicy()
	p.AllowAsset(id)
	reach := &logoReach{policy: p}
	// The deadline is the caller's, not internal/asset's: that room is handed
	// its reach and sets no wire policy, and every other ceiling in this
	// binary is set where the command is.
	ctx, cancel := context.WithTimeout(ctx, logoCeiling)
	defer cancel()
	got, err := asset.Fetch(ctx, reach, id, asset.Want{Mime: cfg.Logo.Mime})
	warns := append([]string{}, reach.opened...)
	if err != nil {
		return nil, logoData{}, warns, err
	}

	data := logoData{Source: got.Source, ID: id, Reason: got.Reason, CacheError: got.CacheError}
	if !got.FetchedAt.IsZero() {
		data.FetchedAt = got.FetchedAt.UTC().Truncate(time.Second).Format(time.RFC3339)
	}
	if got.Age > 0 {
		data.AgeSeconds = int64(got.Age / time.Second)
	}
	if got.Reason != "" {
		warns = append(warns, fmt.Sprintf(
			"the house logo is the copy cached on this machine, %s, because Drive did not answer: %s",
			cacheAge(got), got.Reason))
	}
	if got.CacheError != "" {
		warns = append(warns, fmt.Sprintf(
			"the house logo was read from Drive and could not be cached, so the next build asks again: %s", got.CacheError))
	}
	return got.Raw, data, warns, nil
}

// cacheAge is how old the cached mark is, in the words a person reads. A
// cache with no stamp about it has no age to state, and saying so is better
// than a zero somebody would read as "just now".
func cacheAge(got asset.Bytes) string {
	if got.Age <= 0 {
		return "of an age nothing on this machine records"
	}
	return fmt.Sprintf("fetched %s, %d hours ago", got.FetchedAt.UTC().Format(time.RFC3339), int64(got.Age/time.Hour))
}
