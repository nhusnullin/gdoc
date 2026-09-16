// This file is the one read gdoc makes without a credential. Everything else
// in the package carries Google's bearer; this carries nothing at all.

package gapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gdoc/internal/guard"
)

// ListingTimeout bounds the releases listing. A person typed `gdoc update` and
// is waiting at a terminal, so GitHub not answering has to become an answer
// quickly. The download is not bounded by it: a zip is megabytes on whatever
// connection the machine has, and the caller's own context says how long that
// may take.
const ListingTimeout = 5 * time.Second

// MaxListingBody is the ceiling on the releases listing. The update asks for a
// hundred releases with their assets, which is a few hundred kilobytes, and a
// body without a bound is a memory limit somebody else sets. The ceiling is
// well above the biggest page GitHub will answer, on purpose: it is here to
// stop an answer nothing bounds, not to trim a real one.
const MaxListingBody = 8 << 20

// Plain is one run's reach with no credential: the guard's client, and no
// token anywhere near it.
//
// It exists so that `gdoc update` reads GitHub through the same wire every
// other request goes out on, judged by the same policy, without the bearer
// that every method on Session sets. The separation is the point. A Session
// cannot be pointed at GitHub, because the guard refuses an Authorization
// header on those hosts, and a Plain has nothing to send even if it were
// pointed at Google.
type Plain struct {
	policy *guard.Policy
	client *http.Client
}

// OpenPlain builds the client from p. A base of nil means the real wire.
// Nothing is loaded and nothing can fail, so there is no error to return: a
// reach with no credential has nothing to go wrong before its first request.
func OpenPlain(p *guard.Policy, base http.RoundTripper) *Plain {
	return &Plain{policy: p, client: guard.NewClient(p, base)}
}

// Warnings is the policy's own, and there are no others: this reach refreshes
// nothing and saves nothing.
func (pl *Plain) Warnings() []string {
	return append([]string{}, pl.policy.Warnings()...)
}

// GetJSON reads rawURL within ListingTimeout and decodes the answer into
// `into`.
func (pl *Plain) GetJSON(ctx context.Context, rawURL string, into any) error {
	ctx, cancel := context.WithTimeout(ctx, ListingTimeout)
	defer cancel()
	body, err := pl.get(ctx, rawURL, "application/json", MaxListingBody)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("%s answered 200 with a body that is not JSON: %w", rawURL, err)
	}
	return nil
}

// GetBytes reads rawURL and returns at most limit bytes, on the caller's own
// deadline. It is the download, and the bytes are checked against the
// release's published checksum before anything is replaced.
func (pl *Plain) GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	return pl.get(ctx, rawURL, "*/*", limit)
}

// get is the whole of what this reach does: one GET, no credential, no retry.
// There is no refresh policy here and no 401 to retry, because there is no
// token: a 401 from GitHub would mean gdoc asked for something that is not
// public, and asking a second time answers the same.
func (pl *Plain) get(ctx context.Context, rawURL, accept string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)

	resp, err := pl.client.Do(req)
	if err != nil {
		return nil, unwrapRequestError(err)
	}
	defer resp.Body.Close()

	ok := resp.StatusCode >= 200 && resp.StatusCode <= 299
	read := limit
	if !ok {
		read = maxErrorBody
	}
	// One byte past the ceiling, so a body that fills it can be told from a
	// body that ended, as attempt does for the authenticated reads.
	body, err := io.ReadAll(io.LimitReader(resp.Body, read+1))
	if err != nil {
		return nil, fmt.Errorf("the answer from %s could not be read: %w", rawURL, err)
	}
	if int64(len(body)) > read {
		if !ok {
			return nil, statusError(rawURL, resp.StatusCode, body[:read])
		}
		return nil, fmt.Errorf("the answer from %s is larger than the %d bytes this read allows", rawURL, limit)
	}
	if !ok {
		return nil, statusError(rawURL, resp.StatusCode, body)
	}
	return body, nil
}
