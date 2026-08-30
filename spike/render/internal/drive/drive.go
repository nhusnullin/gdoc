// Package drive is the Drive client, and the guard around it.
//
// The guard is principle 3 in one place: a client reaches only the files it was
// given. In Python that is an httplib2 wrapper; in Go it is an
// http.RoundTripper, which is a smaller and more idiomatic thing to write. It is
// here because a spike that skipped it would leave the hardest constraint
// unproven, not because the rendering needs it.
package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// authorizedUser is the shape google-auth writes with Credentials.to_json.
type authorizedUser struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	TokenURI     string   `json:"token_uri"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes"`
}

// Guard carries a request only when the file it addresses is in its set.
//
// The set has exactly two doors: the ids passed in, and the ids learned from a
// create the guard itself carried. Never add a third, and never widen it to make
// a test pass. A refused call usually means the command did not say which
// document it was for.
type Guard struct {
	inner   http.RoundTripper
	mu      sync.Mutex
	allowed map[string]bool
}

// fileIDInPath finds the file id a Drive URL addresses, if it addresses one.
// A create names no file, which is why it is allowed through and its answer read.
var fileIDInPath = regexp.MustCompile(`/(?:drive/v3|upload/drive/v3)/files/([^/?]+)`)

func NewGuard(inner http.RoundTripper, ids []string) *Guard {
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id != "" {
			allowed[id] = true
		}
	}
	return &Guard{inner: inner, allowed: allowed}
}

func (g *Guard) permit(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.allowed[id] = true
}

func (g *Guard) permitted(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.allowed[id]
}

func (g *Guard) RoundTrip(request *http.Request) (*http.Response, error) {
	if match := fileIDInPath.FindStringSubmatch(request.URL.Path); match != nil {
		if id := match[1]; !g.permitted(id) {
			return nil, fmt.Errorf("refused: this client was never given file %s. "+
				"Name the document the command is for", id)
		}
	}
	response, err := g.inner.RoundTrip(request)
	if err != nil || response == nil {
		return response, err
	}
	// A create comes back carrying the id of the file it made, and that id
	// joins the set. This is the second door, and the only other one.
	if strings.Contains(request.URL.Path, "/files") && request.Method == http.MethodPost {
		g.learn(response)
	}
	return response, nil
}

func (g *Guard) learn(response *http.Response) {
	if response.Header.Get("Content-Type") == "" ||
		!strings.Contains(response.Header.Get("Content-Type"), "json") {
		return
	}
	// Peeking at the body means buffering it. Kept small deliberately: only the
	// id field matters, and a create's response is a few hundred bytes.
	body, err := readAndRestore(response)
	if err != nil {
		return
	}
	var created struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &created) == nil && created.ID != "" {
		g.permit(created.ID)
	}
}

// New builds a Drive client bounded to the given ids.
func New(ctx context.Context, tokenPath string, ids []string) (*drive.Service, error) {
	raw, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("no OAuth token at %s. Run `gdoc auth login` first", tokenPath)
	}
	var stored authorizedUser
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, fmt.Errorf("the token at %s could not be read: %w", tokenPath, err)
	}
	if stored.RefreshToken == "" || stored.ClientID == "" {
		return nil, fmt.Errorf("the token at %s is not an authorised-user token", tokenPath)
	}
	tokenURI := stored.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	config := &oauth2.Config{
		ClientID:     stored.ClientID,
		ClientSecret: stored.ClientSecret,
		Scopes:       stored.Scopes,
		Endpoint:     oauth2.Endpoint{TokenURL: tokenURI},
	}
	source := config.TokenSource(ctx, &oauth2.Token{
		AccessToken:  stored.Token,
		RefreshToken: stored.RefreshToken,
	})

	client := &http.Client{Transport: NewGuard(
		&oauth2.Transport{Source: source, Base: http.DefaultTransport}, ids)}
	return drive.NewService(ctx, option.WithHTTPClient(client))
}
