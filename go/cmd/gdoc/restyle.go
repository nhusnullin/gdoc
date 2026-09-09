// The survey, and nothing else yet.
//
// `gdoc restyle <url> --dry-run` reports what a document holds before anything
// is done to it: its threads with a witness for each, what is pending, its
// chips, its tabs, its named ranges and the revision the reads were made
// against. It writes to no document and to no file.
//
// The apply half is M7b's, and --dry-run is required until it exists. The
// refusal names the milestone rather than the flag alone: a caller told only
// that a flag is missing learns the command is broken, when what is true is
// that the half it wants has not been written yet.
package main

import (
	"context"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/emit"
	"gdoc/internal/restyle"
)

func cmdRestyle(raw []string) emit.Result {
	a, err := parseArgs(raw, flagSet{"--dry-run": false})
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if !a.has("--dry-run") {
		return emit.Result{OK: false, Error: "restyle takes --dry-run and nothing else today: " +
			"this milestone surveys a document and writes to none. The restyle itself is M7b's"}
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	ctx := context.Background()

	// The listing before the Docs read, which is the order every poll in this
	// binary makes its two reads in. A comment written between them is in
	// whichever ran second: listed first, it is a comment the Docs read has not
	// got to yet and a later read places it; read first, it is a comment in the
	// listing with no anchor in a document read a moment before it existed, so
	// the survey reports a thread placed nowhere and says the document holds
	// something it cannot point at.
	//
	// The cursor is nil, because a survey is the whole document rather than a
	// window on it.
	raws, err := comments.Fetch(ctx, r.session, r.id, nil)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		// A failed Docs read is a failed survey, and unlike the export it has no
		// honest partial answer: the chips, the pending suggestions, the named
		// ranges, the tab count and the revision id are all in that one read.
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}

	// The named ranges come out of the read above rather than out of a read of
	// their own. The whole document answers with them, so docs.NamedRangesURL
	// is for the caller that wants only them, which is M7b rechecking the
	// ranges rather than the prose. A second read here would be one more
	// request for facts already in hand.
	//
	// The export is the last of the three, and an export that failed is a
	// warning on a survey that still carries its threads. Survey turns the
	// error into that warning and reports every thread unmatched, exactly as
	// `comments --witness` behaves: the threads are the answer and the witness
	// is a second read on top of them.
	f, exportErr := exportFile(ctx, r)
	report, notes := restyle.Survey(restyle.Input{
		Document:  d,
		Comments:  raws,
		Export:    f,
		ExportErr: exportErr,
	})
	return emit.Result{OK: true, Data: report, Warnings: r.warnings(notes...)}
}
