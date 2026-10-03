// make echoes each recipe line before it runs it, and the build line carries
// the OAuth client secret through -ldflags. On 2026-10-03 a `make build`
// printed it into a terminal and from there into a chat. These tests hold
// that no recipe line carrying the secret is echoed, and that a build still
// says whether the binary it made has one.

package boundary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makefileRecipes(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	var recipes []string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "\t") {
			recipes = append(recipes, line)
		}
	}
	return recipes
}

func TestNoRecipeLineThatCarriesTheClientSecretIsEchoed(t *testing.T) {
	carrying := 0
	for _, line := range makefileRecipes(t) {
		if !strings.Contains(line, "$(LDFLAGS)") && !strings.Contains(line, "GDOC_OAUTH_CLIENT_SECRET") {
			continue
		}
		carrying++
		if !strings.HasPrefix(line, "\t@") {
			t.Errorf("this recipe line carries the client secret and make would print it: %q", strings.TrimSpace(line))
		}
	}
	if carrying == 0 {
		t.Fatal("no recipe line carries the client secret; the Makefile moved and this test did not")
	}
}

func TestABuildSaysWhetherItCarriesASecret(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "SECRET_STATE = client secret $(if $(GDOC_OAUTH_CLIENT_SECRET),set,") {
		t.Error("the Makefile has no SECRET_STATE saying whether GDOC_OAUTH_CLIENT_SECRET was set")
	}
	printed := 0
	for _, line := range makefileRecipes(t) {
		if strings.HasPrefix(line, "\t@echo") && strings.Contains(line, "$(SECRET_STATE)") {
			printed++
		}
	}
	if printed < 2 {
		t.Errorf("%d recipe lines print SECRET_STATE, want one for build and one for dist", printed)
	}
}
