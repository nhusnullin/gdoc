#!/bin/sh
# The gdoc the runner's tests run, and never the real one.
#
# It answers the six argv lists with one JSON object each and tells the test
# what reached it: the variables its environment holds and what its standard
# input gave. No test here ever runs the real binary, because a test that did
# would read somebody's token and ask Google about it.
#
# `update` is the slow one, so the timeout and the one-child-per-kind tests
# have something to stop. It execs sleep rather than sleeping in a subshell,
# so the signal the runner sends reaches the program that holds the pipes and
# nothing is left writing into them.

set -u

# What stdin gave, which is nothing at all when it is the null device.
line=""
eof="yes"
if IFS= read -r line; then
  eof="no"
else
  line=""
fi

path_set="no"
if [ -n "${PATH:-}" ]; then
  path_set="yes"
fi

# The child's own side of the story, as the data object's `child` field.
child() {
  printf '{"home":"%s","path_set":"%s","lang":"%s","no_color":"%s","config_dir":"%s","stdin":"%s","stdin_eof":"%s"}' \
    "${HOME:-}" "$path_set" "${LANG:-}" "${NO_COLOR:-}" "${GDOC_CONFIG_DIR:-}" "$line" "$eof"
}

case "$*" in
"help --json")
  printf '{"ok":true,"version":"v2.14.0","data":{"settings":{"mark":"⁘"},"update":{"installed":"v2.14.0","latest_stable":"v2.14.1"},"child":%s}}\n' "$(child)"
  ;;
"auth status")
  printf '{"ok":true,"version":"v2.14.0","data":{"token_present":true,"expired":false,"missing_scopes":[],"account":"someone@example.com","child":%s}}\n' "$(child)"
  ;;
"auth login")
  printf 'Open this link in your browser to sign in:\n' >&2
  printf 'https://accounts.google.com/o/oauth2/auth?client_id=fake\n' >&2
  printf '{"ok":true,"version":"v2.14.0","data":{"account":"someone@example.com","child":%s}}\n' "$(child)"
  ;;
"auth logout")
  printf '{"ok":true,"version":"v2.14.0","data":{"removed":true,"child":%s}}\n' "$(child)"
  ;;
"update")
  exec sleep 2
  ;;
"update --desktop")
  printf '{"ok":true,"version":"v2.14.0","data":{"installed":"v2.14.0","child":%s}}\n' "$(child)"
  ;;
*)
  printf '{"ok":false,"error":"fake-gdoc.sh was handed an argv it does not answer: %s"}\n' "$*"
  exit 1
  ;;
esac
