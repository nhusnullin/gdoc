package main

import (
	"net/url"
	"strings"
	"testing"
)

// This file is the probe leaving `propose`. Suggestions are generally available,
// so the throwaway document, the create grant and the required folder are gone,
// and the read-backs are what catch a silent direct edit. `--folder` is still
// accepted for one release so a v2.7 caller keeps working, and it does nothing.
// The 2026-10-02 entry in docs/v2/DECISIONS.md holds the decision.

// TestProposeRunsNoProbe is the rule itself. A run touches the document it was
// handed and nothing else: no create, no trash, and no request naming any other
// document.
func TestProposeRunsNoProbe(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)
	if code != 0 || got["ok"] != true {
		t.Fatalf("propose without a folder must run: %v (exit %d)", got, code)
	}
	for _, c := range f.calls {
		if strings.Contains(c.URL, "files?fields=id") {
			t.Errorf("a propose creates no document: %v", c)
		}
		if c.Method == "PATCH" && strings.Contains(string(c.Body), "trashed") {
			t.Errorf("a propose trashes nothing: %v", c)
		}
		if strings.Contains(c.URL, probeDocID) {
			t.Errorf("a propose reaches no document but the one it was handed: %v", c)
		}
	}
	// The probe's verdict was reported beside the proposals. There is no probe,
	// so there is no field: a null verdict would read as a probe that failed.
	if _, has := dataOf(t, got)["probe"]; has {
		t.Error("the envelope must carry no probe")
	}
}

// TestTheFolderFlagIsAcceptedAndIgnored is the one release of grace. The flag is
// still read as a folder id, so a malformed value is refused as before, the run
// says the flag is ignored, and no create door is opened on the policy.
func TestTheFolderFlagIsAcceptedAndIgnored(t *testing.T) {
	withFolder := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from, "--folder", testFolderID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("a folder that is ignored is not a failed run: %v (exit %d)", got, code)
	}
	warns := warningsOf(t, got)
	if !hasWarning(warns, "--folder is ignored") || !hasWarning(warns, "removed in a later release") {
		t.Errorf("warnings = %v, and one must say the flag does nothing and is going", warns)
	}
	if withFolder.policy == nil {
		t.Fatal("the command opened no policy")
	}
	// The create door is the one the probe needed. Asking the policy directly,
	// because a create nothing sends is still a door a later caller could walk
	// through.
	create, err := url.Parse("https://www.googleapis.com/drive/v3/files?fields=id")
	if err != nil {
		t.Fatal(err)
	}
	if err := withFolder.policy.Judge("POST", create, []byte(`{"parents":["`+testFolderID+`"]}`)); err == nil {
		t.Error("the policy still opens a create door in the folder")
	}

	// The same run without the flag sends the same requests, in the same order.
	withoutFolder := stubWire(t, &fakeWire{answers: proposeAnswers(t, true)})
	got, code = runJSON(t, "propose", proposeDocID, "--from", from)
	if code != 0 || got["ok"] != true {
		t.Fatalf("propose: %v (exit %d)", got, code)
	}
	if len(withFolder.calls) != len(withoutFolder.calls) {
		t.Fatalf("the flag changed the run: %d calls with it, %d without", len(withFolder.calls), len(withoutFolder.calls))
	}
	for i := range withFolder.calls {
		a, b := withFolder.calls[i], withoutFolder.calls[i]
		if a.Method != b.Method || a.URL != b.URL || string(a.Body) != string(b.Body) {
			t.Errorf("call %d differs: %v with the flag, %v without", i, a, b)
		}
	}
	if hasWarning(warningsOf(t, got), "--folder") {
		t.Errorf("a run that was given no folder must say nothing about one: %v", warningsOf(t, got))
	}
}

// A folder id is still read, so a document URL in --folder is the mistake it
// always was: the flag is ignored, not unparsed.
func TestProposeStillRefusesAMalformedFolder(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from,
		"--folder", "https://docs.google.com/document/d/"+proposeDocID+"/edit")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a document URL is not a folder: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "document, not a folder") {
		t.Errorf("the error must say which mistake was made: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may be sent for an argument that was refused: %v", f.calls)
	}
}
