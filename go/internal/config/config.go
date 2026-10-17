// Package config decides where gdoc's per-user files live:
// ~/.config/gdoc-agent on macOS, %AppData%\gdoc-agent on Windows.
// GDOC_CONFIG_DIR overrides both, which is also what the test suite uses so
// tests never touch the real config.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

const dirName = "gdoc-agent"

// outside is what Dir reads of the machine it runs on: the platform, and the
// three values the environment supplies.
//
// It exists so dirFor can be a pure function of the case it is handed, which
// is how the Windows branch is checked on a Mac. A branch no test can reach is
// a branch nobody has checked, and this one decides where the OAuth token is
// written.
type outside struct {
	goos      string
	configDir string
	appData   string
	home      string
	homeErr   error
}

// dirFor is the decision, and nothing else. It reads no environment and
// touches no disk.
func dirFor(o outside) (string, error) {
	if o.configDir != "" {
		return o.configDir, nil
	}
	if o.goos == "windows" {
		if o.appData == "" {
			return "", errors.New("config: %AppData% is not set")
		}
		return filepath.Join(o.appData, dirName), nil
	}
	if o.homeErr != nil {
		return "", o.homeErr
	}
	return filepath.Join(o.home, ".config", dirName), nil
}

// Dir is the directory gdoc's per-user files live in. It fails rather than
// guessing: a home directory the OS cannot name would otherwise put the token
// somewhere nobody looks and nobody deletes.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	return dirFor(outside{
		goos:      runtime.GOOS,
		configDir: os.Getenv("GDOC_CONFIG_DIR"),
		appData:   os.Getenv("AppData"),
		home:      home,
		homeErr:   err,
	})
}

// TokenPath is where the OAuth token file sits. The file is oauth-token.json
// in the config dir, in the google-auth "authorized user" shape internal/auth
// reads and writes.
func TokenPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "oauth-token.json"), nil
}

// LastCheckPath is where the record of the last release check sits. The file
// is update-check.json in the config dir, in the shape internal/lastcheck
// reads and writes. It is gdoc's own bookkeeping and never a credential, but
// it lives beside the token because both are per-user state this machine
// keeps.
func LastCheckPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "update-check.json"), nil
}

// LoginPendingPath is where a sign-in waiting for a browser writes itself
// down: login-pending.json in the config dir, in the shape internal/auth
// reads and writes. It sits beside the token because it is about the token:
// two gdoc processes read it so that only one of them opens a listener, and a
// process that finds one does not start a second browser trip.
func LoginPendingPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "login-pending.json"), nil
}

// SettingsPath is where the settings file sits: settings.yaml in the config
// dir, in the shape internal/settings reads. It lives beside the token because
// both are per-user state this machine keeps, and it is never a credential:
// what may be in it is internal/settings' business, and the token is not one
// of the things a colleague may set there.
func SettingsPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "settings.yaml"), nil
}

// assetDir is the folder the cached assets live in, under the config dir. One
// folder rather than one file, because every asset is its own id and there is
// no listing anywhere that would tell them apart from the token beside them.
const assetDir = "assets"

// assetID is what an id may be made of here: Drive's own alphabet and nothing
// else. The shape of a real file id is internal/house's business; this is
// narrower and about one thing only, that the id becomes one path segment.
//
// A dot is refused with the slashes, so no id can be `..` and no stamp can
// collide with the bytes of an id spelled with `.json` at the end.
var assetID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// assetFile is the path of one file under assets/, with suffix on the end.
// The id is checked rather than cleaned: a caller holding an id that is not
// one has read a house file somebody wrote by hand, and the honest answer is
// to name it rather than to write somewhere else.
func assetFile(id, suffix string) (string, error) {
	if !assetID.MatchString(id) {
		return "", fmt.Errorf("config: %q is not a file id, so there is no file under %s for it", id, assetDir)
	}
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, assetDir, id+suffix), nil
}

// AssetPath is where the bytes of one Drive file are cached: assets/<id> in
// the config dir, in the shape internal/asset writes. It lives beside the
// token because both are per-user state this machine keeps, and it is never a
// credential: what is in it is a file the signed-in account may read.
func AssetPath(id string) (string, error) {
	return assetFile(id, "")
}

// AssetStampPath is where the record of the last fetch of that file sits:
// assets/<id>.json, beside the bytes, in the shape internal/asset reads and
// writes. It is the same bookkeeping update-check.json is, for a file rather
// than for a release.
func AssetStampPath(id string) (string, error) {
	return assetFile(id, ".json")
}

// PresenceDir is where the presence record lives: a presence directory in the
// config dir, holding one append-only JSONL file per month in the shape
// internal/presence writes. It is a directory rather than a file because the
// record is kept by month and the oldest months are removed.
//
// It sits beside the token because it is per-user state this machine keeps,
// like update-check.json and login-pending.json, and never beside the hub:
// nothing lives hidden next to somebody's notes. gdoc never reads it; the one
// reader is the presence app.
func PresenceDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "presence"), nil
}
