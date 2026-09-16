// Package auth holds the OAuth token, its refresh, and the login flow.
//
// Two things live here: the token file the whole binary signs its requests
// from, and the client id and secret gdoc ships with. The token endpoint is two
// form POSTs, the refresh and the code exchange, which is why x/oauth2 is not a
// dependency. This package never builds an HTTP client: it is handed one, and
// internal/guard is the only room that builds one, so every POST here is judged
// like every other request gdoc makes.
//
// This comment holds why the package refuses what it refuses, with the test
// that pins each rule. What the rules are is in the code beside them.
//
// # The token file is google-auth's shape, and a token already on disk is a login
//
// Load reads oauth-token.json from the config dir in google-auth's "authorized
// user" shape, field names included. The shape is read rather than invented, so
// a token written by the Python tool this binary replaces needs no migration
// and no second browser trip. TestLoadReadsTheAuthorizedUserShape is the pin.
//
// # The login asks for the read/write Docs scope
//
// loginScopes is the full Drive scope plus documents, which is read/write.
// Not documents.readonly: gdoc writes suggestions through the Docs API, and
// the read-only scope cannot carry a batchUpdate.
//
// The consequence runs one way, and it has to stay written down. A token this
// login writes records documents and not documents.readonly, so a reader that
// asks for the read-only scope by name does not find it in the file and asks
// for its own sign-in. The other direction costs nothing, because the Drive
// scope covers the Docs calls: read the next rule. The two sets are not equal,
// and a comment in login.go that calls them equal is wrong.
//
// # MissingScopes is a report, and the full Drive scope covers the Docs calls
//
// The Docs API accepts auth/drive on documents.get and documents.batchUpdate,
// so a token holding Drive plus documents.readonly is missing nothing gdoc
// needs. Comparing the requested list literally warned on a working token, and
// a warning on the working case is one people learn to ignore. coveredBy in
// login.go is where that lives. drive.file is deliberately not in it: that
// scope reaches only files the app itself created.
//
// It is a report, never a refusal. A partial grant comes from Google's granular
// consent screen, where a person ticks a subset, and then the Docs calls will
// 403. This is the one place that can say why before they do.
// TestMissingScopesReadsDriveAsCoveringDocs and
// TestStatusIsQuietForATokenCarryingTheFullDriveScope are the pins, and the
// second is the working case: it fails if a token that works ever starts
// carrying a warning.
//
// # What the file records is what was granted, never what was asked for
//
// grantedScopes reads the token endpoint's own scope field. Recording the
// requested list instead would let auth status report scopes the token does not
// have, and a later 403 would have nothing in the file to explain it. Silence
// means the grant matched the request, which is the only case RFC 6749 section
// 5.1 makes the field optional in, so an absent scope falls back to the
// requested set. TestExchangeRecordsTheGrantedScopesNotTheRequestedOnes and
// TestExchangeFallsBackToTheRequestedScopesWhenTheEndpointSaysNothing are the
// pins.
//
// # Save carries three fields gdoc never uses
//
// universe_domain, account and rapt_token go through untouched. They are
// google-auth's fields, Credentials.to_json writes all three when they are set,
// gdoc uses none of them, and dropping one would quietly rewrite a file another
// program shares. rapt_token is the reauth proof token, so losing it makes that
// program ask for reauthentication again. TestSaveKeepsFieldsV2DoesNotUse is
// the pin.
//
// The write goes through internal/atomicfile at 0o600, whatever mode the file
// carried before: it is a credential, and a failed write must not leave the
// token that worked truncated. TestSaveIsCrashSafe,
// TestFailedSaveLeavesTheOriginalIntact and TestSaveWritesAPrivateFile are the
// pins.
//
// # Load refuses a file that parses but cannot be refreshed
//
// The required set is google-auth's, not one gdoc invented:
// from_authorized_user_info raises without refresh_token, client_id and
// client_secret. A {} that read as a token would have auth status report a
// token present and the first Docs call fail with something else. Only an
// absent file means signed out, which is ErrNoToken and its own error for that
// reason. TestLoadRefusesATokenMissingTheFieldsARefreshNeeds,
// TestLoadRefusesAnUnreadableToken and TestLoadWithNoTokenNamesTheFileAndTheFix
// are the pins.
//
// token_uri is not in that set, because google-auth overrides it with its own
// constant whatever the file says. Load fills the same value in rather than
// posting a refresh to an empty URL.
// TestLoadFillsInTheTokenEndpointWhenTheFileOmitsIt is the pin.
//
// # A code exchange with no refresh token is not a login
//
// The access token would work for about an hour, then every refresh would post
// an empty refresh_token and fail for good, with no way back except a fresh
// login, and Save would already have written over the token that did work. So
// exchangeCode refuses it and nothing is saved. The request carries
// access_type=offline with prompt=consent, so a missing refresh token is an
// anomaly rather than a normal reply. TestExchangeRefusesA200WithNoRefreshToken
// and TestARefreshlessExchangeDoesNotOverwriteAGoodToken are the pins.
//
// # The client id is shipped, and the secret is injected at build time
//
// BundledClientID is a constant: a client id is public in every sign-in URL,
// and nobody using gdoc visits a cloud console. BundledClientSecret is a
// variable that is empty in the source and set by the linker from
// GDOC_OAUTH_CLIENT_SECRET in make build and make dist, so a release build
// carries it and the tree never does. RFC 8252 section 8.5 still holds, a
// secret shipped to many users "should not be treated as confidential", and
// what protects an account is still the per-user token, which never leaves the
// machine. What changed on 2026-09-16 is where the secret lives, not what it
// protects: the repository went public that day, and GitHub reports a Google
// client secret it finds in a public commit to Google, who may revoke the
// client and break every colleague's login at once. The secret that had been
// in the tree was rotated the same day. Decided 2026-08-18 and amended
// 2026-09-16, docs/v2/DECISIONS.md.
//
// A build without the secret can refresh a token it already holds, because
// the token file carries the secret it was issued with, but it cannot sign
// anyone in, and Login refuses before it opens a listener or prints a URL.
// TestLoginRefusesABuildWithNoClientSecret is the pin, and
// TestNoGoogleClientSecretInTheTree in go/boundary holds the other half: no
// file in the tree carries a string shaped like a Google client secret.
//
// # The client must stay User type Internal
//
// gdoc needs the full Drive scope, which Google classes as restricted. Internal
// exempts gdoc from OAuth verification, from the unverified-app screen and from
// the 100-user cap. External would mean a CASA assessment every 12 months, and
// refresh tokens expiring weekly.
//
// TODO(test): no test pins the client type. It is about the Google project
// rather than about code, so there is nothing in the tree for a test to read.
//
// # The per-user client override is read by nothing yet
//
// clientFileName names oauth-client.json in the config dir. The Python tool
// this binary replaces lets that file override the bundled client, which is a
// quota override rather than a setup step: one shared client shares one Google
// rate limit. Login here does not read it. Status says so instead, and
// cmd/gdoc's package comment holds what it reports. Saying the client came from
// a file would name a client no token was ever issued to.
// TestStatusSaysAClientFileIsNotUsedYet is the pin.
package auth
