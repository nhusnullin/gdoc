package update

import "testing"

// TestThePolicyTable walks the table in the M9 plan, row for row. Each row is
// an installed version, what each channel holds, and the flags a person typed,
// and the answer is what `gdoc update` does about it. The versions are
// literals here, because a policy that reads its own constants follows them
// wherever somebody moves them.
func TestThePolicyTable(t *testing.T) {
	cases := []struct {
		name      string
		installed string
		stable    string
		nightly   string
		flags     Flags
		action    Action
		to        string
		run       string
	}{
		{
			name:      "a nightly is available and the stable channel has nothing new",
			installed: "v2.0.0", stable: "v2.0.0", nightly: "v2.0.3",
			action: UpToDate, run: "gdoc update --nightly",
		},
		{
			name:      "a new stable is taken, and the higher nightly is not",
			installed: "v2.0.0", stable: "v2.1.0", nightly: "v2.1.2",
			action: Updated, to: "v2.1.0",
		},
		{
			name:      "a major is named and never taken unasked",
			installed: "v2.0.0", stable: "v3.0.0",
			action: MajorAvailable, to: "v3.0.0", run: "gdoc update --major",
		},
		{
			name:      "a major named on the nightly channel names the nightly command",
			installed: "v2.1.0", stable: "v2.1.0", nightly: "v3.0.1", flags: Flags{Nightly: true},
			action: MajorAvailable, to: "v3.0.1", run: "gdoc update --major --nightly",
		},
		{
			name:      "a nightly run across a major takes it once the person asks",
			installed: "v2.1.0", stable: "v2.1.0", nightly: "v3.0.1", flags: Flags{Nightly: true, Major: true},
			action: Updated, to: "v3.0.1",
		},
		{
			name:      "the major the person asked for is taken",
			installed: "v2.0.0", stable: "v3.0.0", flags: Flags{Major: true},
			action: Updated, to: "v3.0.0",
		},
		{
			name:      "the nightly the person asked for is taken",
			installed: "v2.0.0", stable: "v2.0.0", nightly: "v2.0.3", flags: Flags{Nightly: true},
			action: Updated, to: "v2.0.3",
		},
		{
			name:      "a nightly run takes the stable when the stable is the higher of the two",
			installed: "v2.0.3", stable: "v2.1.0", nightly: "v2.1.0", flags: Flags{Nightly: true},
			action: Updated, to: "v2.1.0",
		},
		{
			name:      "never down",
			installed: "v2.1.0", stable: "v2.0.0",
			action: UpToDate,
		},
		{
			name:      "never down, whatever was typed",
			installed: "v2.1.0", stable: "v2.0.0", flags: Flags{Major: true, Nightly: true},
			action: UpToDate,
		},
		{
			name:      "a check writes nothing and names what it would have taken",
			installed: "v2.0.0", stable: "v2.1.0", nightly: "v2.1.2", flags: Flags{Check: true},
			action: Checked, to: "v2.1.0",
		},
		{
			name:      "a check on a major names the major and the command for it",
			installed: "v2.0.0", stable: "v3.0.0", flags: Flags{Check: true},
			action: Checked, to: "v3.0.0", run: "gdoc update --major",
		},
		{
			name:      "a check with nothing to take names nothing to take",
			installed: "v2.1.0", stable: "v2.1.0", flags: Flags{Check: true},
			action: Checked,
		},
		{
			name:      "a check on the nightly channel names the nightly",
			installed: "v2.0.0", stable: "v2.0.0", nightly: "v2.0.3", flags: Flags{Check: true, Nightly: true},
			action: Checked, to: "v2.0.3",
		},
		{
			name:      "a channel nothing was found in takes nothing",
			installed: "v2.0.0",
			action:    UpToDate,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Decide(State{
				Installed: mustParse(t, c.installed),
				Stable:    parseOrZero(t, c.stable),
				Nightly:   parseOrZero(t, c.nightly),
			}, c.flags)
			if got.Action != c.action {
				t.Errorf("action = %q, want %q", got.Action, c.action)
			}
			if got.To.String() != c.to {
				t.Errorf("to = %q, want %q", got.To, c.to)
			}
			if got.Run != c.run {
				t.Errorf("run = %q, want %q", got.Run, c.run)
			}
		})
	}
}

func TestOnlyAnUpdatedDecisionInstalls(t *testing.T) {
	// The command reads this one field to decide whether anything is
	// downloaded at all, so a check or a version that is already the latest
	// touches no file.
	installs := Decide(State{Installed: mustParse(t, "v2.0.0"), Stable: mustParse(t, "v2.1.0")}, Flags{})
	if !installs.Installs() {
		t.Error("a decision to update does not report that it installs")
	}
	for _, d := range []Decision{
		Decide(State{Installed: mustParse(t, "v2.0.0"), Stable: mustParse(t, "v2.1.0")}, Flags{Check: true}),
		Decide(State{Installed: mustParse(t, "v2.1.0"), Stable: mustParse(t, "v2.1.0")}, Flags{}),
		Decide(State{Installed: mustParse(t, "v2.0.0"), Stable: mustParse(t, "v3.0.0")}, Flags{}),
	} {
		if d.Installs() {
			t.Errorf("a %q decision reports that it installs", d.Action)
		}
	}
}

func TestTheChannelADecisionReadsIsTheOneTheFlagNamed(t *testing.T) {
	if got := (Flags{}).Channel(); got != Stable {
		t.Errorf("Channel() without --nightly = %v, want Stable", got)
	}
	if got := (Flags{Nightly: true}).Channel(); got != Nightly {
		t.Errorf("Channel() with --nightly = %v, want Nightly", got)
	}
}

func parseOrZero(t *testing.T, s string) Version {
	t.Helper()
	if s == "" {
		return Version{}
	}
	return mustParse(t, s)
}
