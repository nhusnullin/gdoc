// Package config decides where gdoc's per-user files live. v1's macOS path
// (~/.config/gdoc-agent) is preserved on purpose; %AppData%\gdoc-agent on
// Windows. GDOC_CONFIG_DIR overrides both, which is also what the test suite
// uses so tests never touch the real config.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

const dirName = "gdoc-agent"

func Dir() (string, error) {
	if d := os.Getenv("GDOC_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if runtime.GOOS == "windows" {
		appData := os.Getenv("AppData")
		if appData == "" {
			return "", errors.New("config: %AppData% is not set")
		}
		return filepath.Join(appData, dirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", dirName), nil
}

func TokenPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "oauth-token.json"), nil
}
