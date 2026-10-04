// The daily release check, and the one line it prints.
//
// `gdoc help` and `gdoc mcp` are the two commands that ask GitHub what is
// published without being told to. help is the first call of every skill
// session, and mcp is a chat where nobody types a command at all, so its first
// tool answer carries the line instead: mcp.go's mcpNotice. Every other command
// is exactly as offline from GitHub as it ever was.
//
// The check costs a run nothing it can feel. It reads one small file; the
// fetch behind it runs only when that file is missing or older than a day,
// under a ceiling of two seconds, and a failed fetch is written down too, so a
// machine that cannot reach GitHub pays the ceiling once a day rather than
// once a run. A build from a checkout names no release, so it has nothing to
// compare and never asks at all.

package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"gdoc/internal/config"
	"gdoc/internal/guard"
	"gdoc/internal/lastcheck"
	"gdoc/internal/update"
)

// checkTimeout is the whole of what the check may cost a help. The listing has
// its own five-second bound inside gapi, for a person who typed `gdoc update`
// and is waiting for an answer; this one is a fetch nobody asked for, so it
// gets less. The earlier deadline wins, so two seconds is what holds.
const checkTimeout = 2 * time.Second

// updateFacts is what help says about releases: what is installed, what is
// published on each channel, when gdoc last asked, and what stopped it if
// something did. Facts only. Nothing here says "available", "behind" or
// "should": the skills and the person reading them decide that.
type updateFacts struct {
	Installed     string `json:"installed,omitempty"`
	LatestStable  string `json:"latest_stable,omitempty"`
	LatestNightly string `json:"latest_nightly,omitempty"`
	CheckedAt     string `json:"checked_at,omitempty"`
	Error         string `json:"error,omitempty"`
}

// notice is the whole of the check: the facts for the object, the one line a
// person reads, and the warnings for the envelope. It writes nothing. Whoever
// asked decides whether the line reaches a stream, which is what lets a caller
// that has no stream to print to still have the facts.
//
// A checkout build returns nothing, which is what keeps `make build` binaries
// and the whole test suite off the network.
//
// TestNoticeReturnsTheLine, TestAStaleStampMakesHelpAskOnce,
// TestAFreshStampMakesNoRequest, TestACheckoutBuildNeverChecks and
// TestReadNeverReachesTheCheck.
func notice(ctx context.Context) (*updateFacts, string, []string) {
	installed := releaseVersion()
	if installed == "" {
		return nil, "", nil
	}
	path, err := config.LastCheckPath()
	if err != nil {
		return nil, "", []string{fmt.Sprintf("gdoc cannot say where it keeps the record of the last release check, so it did not ask whether there is a newer one: %v", err)}
	}

	var warns []string
	// Why the stamp is stale is dropped rather than reported: asking GitHub
	// again is the whole answer, and the ordinary case is the first run on a
	// machine, where a warning about a file nobody has written yet is noise.
	stamp, stale, _ := lastcheck.Read(path, time.Now())
	if stale {
		var checkWarns []string
		stamp, checkWarns = ask(ctx, stamp)
		warns = append(warns, checkWarns...)
		// A stamp that cannot be written is one warning and no second
		// attempt: the check already happened, and a run that asked twice
		// would cost what this whole design exists to bound.
		if err := lastcheck.Write(path, stamp); err != nil {
			warns = append(warns, fmt.Sprintf("gdoc asked what is published but could not write the answer to %s, so it will ask again on the next help: %v", path, err))
		}
	}

	return &updateFacts{
		Installed:     installed,
		LatestStable:  stamp.LatestStable,
		LatestNightly: stamp.LatestNightly,
		CheckedAt:     checkedAt(stamp),
		Error:         stamp.Error,
	}, noticeLine(installed, stamp), warns
}

// ask is the read itself: the fifth grant for one run, one GET on the releases
// listing, and the newest release of each channel out of the answer.
//
// It is runUpdate's read without the decision, so it shares the repository,
// the URL and the choosing, and it installs nothing and touches no binary. The
// stamp it answers with is always dated now, whether the read worked or not,
// because what it records is when gdoc asked.
//
// TestTheHelpCheckOpensThePolicyTheUpdateOpens,
// TestTheCheckIsBoundedByTwoSeconds and
// TestAnUnreachableGitHubIsStampedAndHelpStillAnswers.
func ask(ctx context.Context, prev lastcheck.Stamp) (lastcheck.Stamp, []string) {
	// The versions heard before are kept, so a check that fails today still
	// leaves help able to say what it knew yesterday.
	next := lastcheck.Stamp{
		CheckedAt:     time.Now(),
		LatestStable:  prev.LatestStable,
		LatestNightly: prev.LatestNightly,
	}

	p := guard.NewPolicy()
	p.AllowUpdateFrom(updateRepo)
	reach := openPlain(p)

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	var entries []update.Entry
	if err := reach.GetJSON(ctx, releasesURL, &entries); err != nil {
		next.Error = fmt.Sprintf("the releases of %s could not be read, so this check cannot say whether there is a newer gdoc: %v", updateRepo, err)
		return next, []string{next.Error}
	}

	platform := update.Platform(runtime.GOOS, runtime.GOARCH)
	var causes []string
	if stable, err := update.Choose(entries, update.Stable, platform); err != nil {
		causes = append(causes, err.Error())
	} else {
		next.LatestStable = stable.Version.String()
	}
	if nightly, err := update.Choose(entries, update.Nightly, platform); err != nil {
		causes = append(causes, err.Error())
	} else {
		next.LatestNightly = nightly.Version.String()
	}
	if len(causes) == 0 {
		return next, nil
	}
	next.Error = fmt.Sprintf("the releases of %s were read and a channel had nothing this machine could install: %s", updateRepo, joinCauses(causes))
	return next, []string{next.Error}
}

// joinCauses puts two causes in one sentence without a list, because a warning
// is read as prose and there are never more than two.
func joinCauses(causes []string) string {
	if len(causes) == 1 {
		return causes[0]
	}
	return causes[0] + "; " + causes[1]
}

// noticeLine is the one line a person reads, and the empty string when there
// is nothing to say. The judgement is update.Decide over the stamp's versions
// with no flags typed, which is the same arithmetic `gdoc update --check`
// does, so the line and the command cannot disagree.
//
// A version that does not parse is no line at all. This binary names a release
// or it named nothing, and guessing at half a comparison is worse than silence.
//
// TestNoticeReturnsTheLine, TestAStaleStampMakesHelpAskOnce and
// TestTheLineNamesMajorForAMajor.
func noticeLine(installed string, s lastcheck.Stamp) string {
	d, ok := noticeDecision(installed, s.LatestStable, s.LatestNightly)
	if !ok {
		return ""
	}
	switch d.Action {
	case update.Updated:
		return fmt.Sprintf("gdoc %s is published and this is %s. `gdoc update` installs it.", d.To, installed)
	case update.MajorAvailable:
		return fmt.Sprintf("gdoc %s is published and this is %s. It is a major release: `gdoc update --major` installs it.", d.To, installed)
	}
	return ""
}

// mcpNoticeLine is that same line for a chat, where there is no terminal in
// front of the reader and a new binary only reaches the session that starts
// after it. A toggle of the connector restarts the chat process and leaves
// agent mode on the old binary (docs/v2/MEASURED.md, measurement 2), so the
// words are quit and open again, and never toggle.
//
// It is built from the versions alone, like noticeLine, and never from a URL in
// the listing: a line a person is told to act on says a command they can read.
//
// TestAStaleStampAsksOnceAndTheFirstAnswerCarriesTheLine and
// TestTheLineSaysQuitAndOpenAgainNeverToggle.
func mcpNoticeLine(f *updateFacts) string {
	if f == nil {
		return ""
	}
	d, ok := noticeDecision(f.Installed, f.LatestStable, f.LatestNightly)
	if !ok {
		return ""
	}
	switch d.Action {
	case update.Updated:
		return fmt.Sprintf("gdoc %s is published and this is %s. Run gdoc update in a terminal, then quit Claude Desktop and open it again.", d.To, f.Installed)
	case update.MajorAvailable:
		return fmt.Sprintf("gdoc %s is published and this is %s. It is a major release: run gdoc update --major in a terminal, then quit Claude Desktop and open it again.", d.To, f.Installed)
	}
	return ""
}

// noticeDecision is the arithmetic both lines are built from: update.Decide
// over the parsed versions with no flags typed, which is what
// `gdoc update --check` does, so neither line can disagree with the command.
//
// It answers false where a version does not parse. This binary names a release
// or it named nothing, and guessing at half a comparison is worse than silence.
func noticeDecision(installed, stable, nightly string) (update.Decision, bool) {
	here, err := update.Parse(installed)
	if err != nil {
		return update.Decision{}, false
	}
	latest, err := update.Parse(stable)
	if err != nil {
		return update.Decision{}, false
	}
	nightliest, err := update.Parse(nightly)
	if err != nil {
		nightliest = update.Version{}
	}
	return update.Decide(update.State{Installed: here, Stable: latest, Nightly: nightliest}, update.Flags{}), true
}

// checkedAt is the stamp's time as the object prints it, and empty when there
// is none: a field saying the zero time would be a fact nobody measured.
func checkedAt(s lastcheck.Stamp) string {
	if s.CheckedAt.IsZero() {
		return ""
	}
	return s.CheckedAt.UTC().Truncate(time.Second).Format(time.RFC3339)
}
