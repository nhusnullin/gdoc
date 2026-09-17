// The daily release check, and the one line it prints.
//
// `gdoc help` is the only command that asks GitHub what is published without
// being told to, because it is the first call of every skill session, it
// already carries the version, and no document is open in front of it. Every
// other command is exactly as offline from GitHub as it ever was.
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
	"io"
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

// notice is the whole of the check: the facts for the object, the warnings for
// the envelope, and at most one line on errOut for the person reading.
//
// A checkout build returns nothing, which is what keeps `make build` binaries
// and the whole test suite off the network.
//
// TestAStaleStampMakesHelpAskOnce, TestAFreshStampMakesNoRequest,
// TestACheckoutBuildNeverChecks and TestReadNeverReachesTheCheck.
func notice(ctx context.Context, errOut io.Writer) (*updateFacts, []string) {
	installed := releaseVersion()
	if installed == "" {
		return nil, nil
	}
	path, err := config.LastCheckPath()
	if err != nil {
		return nil, []string{fmt.Sprintf("gdoc cannot say where it keeps the record of the last release check, so it did not ask whether there is a newer one: %v", err)}
	}

	var warns []string
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

	if line := noticeLine(installed, stamp); line != "" {
		fmt.Fprint(errOut, line+"\n\n")
	}
	return &updateFacts{
		Installed:     installed,
		LatestStable:  stamp.LatestStable,
		LatestNightly: stamp.LatestNightly,
		CheckedAt:     checkedAt(stamp),
		Error:         stamp.Error,
	}, warns
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
// TestAStaleStampMakesHelpAskOnce and TestTheLineNamesMajorForAMajor.
func noticeLine(installed string, s lastcheck.Stamp) string {
	here, err := update.Parse(installed)
	if err != nil {
		return ""
	}
	stable, err := update.Parse(s.LatestStable)
	if err != nil {
		return ""
	}
	nightly, err := update.Parse(s.LatestNightly)
	if err != nil {
		nightly = update.Version{}
	}
	d := update.Decide(update.State{Installed: here, Stable: stable, Nightly: nightly}, update.Flags{})
	switch d.Action {
	case update.Updated:
		return fmt.Sprintf("gdoc %s is published and this is %s. `gdoc update` installs it.", d.To, installed)
	case update.MajorAvailable:
		return fmt.Sprintf("gdoc %s is published and this is %s. It is a major release: `gdoc update --major` installs it.", d.To, installed)
	}
	return ""
}

// checkedAt is the stamp's time as the object prints it, and empty when there
// is none: a field saying the zero time would be a fact nobody measured.
func checkedAt(s lastcheck.Stamp) string {
	if s.CheckedAt.IsZero() {
		return ""
	}
	return s.CheckedAt.UTC().Truncate(time.Second).Format(time.RFC3339)
}
