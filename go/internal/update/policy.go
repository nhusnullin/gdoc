package update

// Action is what a run of `gdoc update` did, or decided not to do. It is one
// word in the object the command prints, and it is a fact: the skills and the
// person reading them decide what it means.
type Action string

const (
	// UpToDate is the binary on disk already being the newest of its channel.
	UpToDate Action = "up_to_date"
	// Updated is a new binary in place, verified.
	Updated Action = "updated"
	// MajorAvailable is a newer release across a major boundary, which is
	// never taken without the flag that names it.
	MajorAvailable Action = "major_available"
	// Checked is `--check`: what was found, and nothing written.
	Checked Action = "checked"
	// Unreachable is GitHub not answering. It is not a failure, and the
	// command sets it rather than Decide, which is handed versions and never
	// a wire.
	Unreachable Action = "unreachable"
	// RolledBack is `--rollback`: the previous binary back in place. It does
	// not depend on any version, so Decide never returns it.
	RolledBack Action = "rolled_back"
)

// Flags are the four words `gdoc update` takes, as the command parsed them.
type Flags struct {
	Check    bool
	Major    bool
	Nightly  bool
	Rollback bool
}

// Channel is the one the flags named.
func (f Flags) Channel() Channel {
	if f.Nightly {
		return Nightly
	}
	return Stable
}

// State is what a run knows before it decides: the version of the binary that
// is running, and the newest release of each channel. A channel nothing was
// found in is the zero Version.
type State struct {
	Installed Version
	Stable    Version
	Nightly   Version
}

// Decision is what the run does about that state.
type Decision struct {
	// Action is one of the six words above.
	Action Action
	// To is the version this run would install, and the zero Version when
	// there is nothing to install. A Checked decision carries it too, which is
	// the whole point of asking.
	To Version
	// Run is the command that would go further than this one did, and empty
	// when there is none. A major needs --major, and a nightly ahead of the
	// stable channel needs --nightly.
	Run string
}

// Installs reports whether anything is downloaded and replaced. It is the one
// field the command reads to decide that, so a check and an up-to-date binary
// take the same path: none.
func (d Decision) Installs() bool { return d.Action == Updated }

// Decide is the policy table in the M9 plan, as code. It is handed versions
// and returns a decision; it reads no file, opens no connection and holds no
// state, so every row of that table is a test over literals.
//
// The rules, in the order they apply:
//
//   - The target is the newest release of the channel the flags named. On the
//     nightly channel that is the newest release there is, which on the day a
//     stable is cut is the stable.
//   - gdoc never goes down. A target at or below what is installed is
//     up to date, whatever was typed.
//   - A target across a major boundary is named and not taken, unless --major
//     was typed. A major is where something a person relies on may have gone.
//   - A stable run that is up to date, with a newer nightly behind it, says so
//     and names the command that would take it.
//   - --check turns whatever the above decided into Checked, keeping the
//     version it would have taken. Nothing else changes, so a check is an
//     honest rehearsal of the run without it.
func Decide(s State, f Flags) Decision {
	d := decide(s, f)
	if f.Check {
		d.Action = Checked
	}
	return d
}

func decide(s State, f Flags) Decision {
	target := s.Stable
	if f.Channel() == Nightly && Compare(s.Nightly, target) > 0 {
		target = s.Nightly
	}
	if target.IsZero() || Compare(target, s.Installed) <= 0 {
		return Decision{Action: UpToDate, Run: nightlyRun(s, f)}
	}
	if target.Major > s.Installed.Major && !f.Major {
		return Decision{Action: MajorAvailable, To: target, Run: "gdoc update --major"}
	}
	return Decision{Action: Updated, To: target}
}

// nightlyRun names the nightly command when a stable run has nothing to take
// and the nightly channel does. On a nightly run there is nothing further to
// name, because the nightly channel is already everything there is.
func nightlyRun(s State, f Flags) string {
	if f.Channel() == Nightly {
		return ""
	}
	if Compare(s.Nightly, s.Installed) > 0 {
		return "gdoc update --nightly"
	}
	return ""
}
