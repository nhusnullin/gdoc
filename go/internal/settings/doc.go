// Package settings reads what a colleague set, from the file and from the
// command line, and answers what this run writes.
//
// It is a package of its own rather than part of internal/config because the
// two answer different questions. config answers where a per-user file lives,
// and internal/auth and internal/lastcheck import it for that. settings
// decodes YAML and asks internal/plaintext what a mark may be, and neither of
// those belongs under the token's feet.
//
// This comment holds why the package decides what it decides. What it reports
// is in the code beside it.
//
// # The order is the argument, then the file, then the default
//
// Resolve lays the three over each other in that order, which is decision B of
// the v2 milestone 16 specification. The built-in default is
// plaintext.DefaultMark, so the mark a new run writes is the quieter one and
// nothing here decides which character that is.
//
// Each field of Given is a pointer, because --mark "" is the empty mark a
// colleague asked for and not the absence of an argument. InEffect carries
// MarkFrom beside the mark so help --json can print where the value came from:
// TestNoFileIsEveryDefault, TestTheFileSetsTheMark,
// TestASettingArgumentOverridesTheFile and TestMarkFromSaysWhichSourceWon.
//
// The file is read whether or not an argument overrides what it says. A file
// gdoc cannot make sense of is a file the colleague has to fix either way, and
// a refusal that depended on which flags were typed would hide it until the
// day they were not.
//
// # The file is decoded strictly, and a named file that is absent is refused
//
// An unknown key is refused by name, as internal/frontmatter and
// internal/house decode their own files. A key gdoc does not know is a setting
// somebody believes is in effect, and every default returned over it would be
// a silent no: TestAnUnknownKeyIsRefusedByName.
//
// The default file absent is every default, because a machine nobody has
// configured is the common case. A file --config named and that is not there is
// refused, because a mistyped path would otherwise hand back every default and
// say nothing: TestAConfigFileThatDoesNotExistIsRefused.
//
// A file of zero bytes is every default, and it is still the file in use, so
// InEffect.File names it: TestAFileOfNoBytesIsEveryDefault.
//
// # mark: "" is the empty mark, and mark: alone is unfinished
//
// The empty mark is a choice, and it writes the words alone. A key with nothing
// behind it is a line somebody did not finish, so it is refused with the way to
// ask for the empty mark rather than read as one:
// TestAnEmptyMarkInTheFileIsTheEmptyMark and TestAMarkKeyWithNoValueIsRefused.
//
// The mark from either source goes through plaintext.CheckMark, which owns the
// whole rule about what a mark may be. The refusal names where the value came
// from, the file and its key or --mark, because that is the line a colleague
// has to go and change:
// TestABadMarkIsRefusedNamingWhereItCameFrom.
//
// # The token is not a setting
//
// Neither the token, nor its path, nor the update stamp. They are a credential
// and gdoc's own bookkeeping, and a file a colleague edits by hand is not where
// a credential's location is decided: a settings file that could move the token
// path is a settings file that can point gdoc at somebody else's token. Both
// words read as unknown keys, which is the refusal any other word gets:
// TestTheTokenIsNotASetting.
package settings
