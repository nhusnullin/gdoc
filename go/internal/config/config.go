// Package config decides where gdoc's per-user files live:
// ~/.config/gdoc-agent on macOS, %AppData%\gdoc-agent on Windows.
// GDOC_CONFIG_DIR overrides both, which is also what the test suite uses so
// tests never touch the real config.
package config

import (
	"errors"
	"os"
	"path/filepath"
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
