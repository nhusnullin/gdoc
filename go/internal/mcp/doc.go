// Package mcp is the Model Context Protocol over stdio, written by hand: one
// JSON-RPC message per line in, one compact line out, until the reader ends.
// It is what `gdoc mcp` serves to Claude Desktop.
//
// It is the protocol and nothing else. It takes a list of tools, each a name, a
// title, a description, a schema and a function, and it knows no command, no
// document, no grant and no hold. It imports no net/http and builds no client:
// TestMcpImportsNoNetHTTP in go/boundary holds that, in both directions.
// docs/v2/DECISIONS.md holds the 2026-10-03 decision this package implements,
// and docs/plans/2026-10-02-gdoc-v2-m14-chat.md the specification.
//
// Nothing here is a dependency. There is a Go SDK for this protocol and gdoc
// has three modules, each named in go/boundary with its reason. Four methods
// and one line format is less code than the argument for a fourth would be.
//
// # The handshake
//
// initialize answers the protocol version, the capabilities, the server info
// and the instructions, in that order, and
// TestInitializeCarriesCapabilitiesInfoAndInstructions pins those bytes.
//
// The versions gdoc speaks are 2025-11-25, 2025-06-18, 2025-03-26 and
// 2024-11-05, newest first. The client's version is echoed when it is one of
// them and the newest is answered otherwise, so a client ahead of gdoc still
// gets a version it can read: TestInitializeAnswersEachSupportedVersion and
// TestAnUnsupportedVersionGetsTheNewest. A server with tools only sends the
// same shapes under all four, which is why the list is the whole of what
// negotiation amounts to here.
//
// The capabilities say tools, with listChanged true, because a hold registers
// its own one-time confirm tool while the session runs.
//
// An empty version is a checkout build, which belongs to no release, and says
// dev rather than naming one: TestAnEmptyVersionIsDev. That is the same rule
// internal/emit follows when it leaves the envelope's version out.
//
// # The methods
//
// initialize, notifications/initialized, tools/list, tools/call and ping.
// Anything else is method not found, -32601, before initialize as readily as
// after, so a client that probes with server/discover falls back cleanly:
// TestAnUnknownMethodIsMethodNotFoundBeforeAndAfterInitialize.
//
// A notification is never answered, known or unknown, because there is nothing
// to answer it to: TestANotificationGetsNoAnswer and
// TestAnUnknownNotificationIsIgnored.
//
// # The framing
//
// The id is kept as raw JSON and echoed byte for byte, string or number:
// TestStringAndNumberIDsAreEchoedExactly, which carries "7", 7, 7.0 and a
// number too long for a float. A message with no id key is a notification;
// id null is not, so what is recorded is the key's presence and never the
// value's emptiness.
//
// A line that is not JSON is a parse error, -32700, with id null, and reading
// goes on. A line that is valid JSON but not an object, and a line that starts
// with a bracket, are each one invalid request, -32600: batches are gone from
// this protocol. A line over maxLine is answered as an invalid request and
// thrown away as it is read, so the reader lands on the next newline rather
// than growing until the machine stops.
// TestAMalformedLineAnOversizeLineAndABatchEachGetOneErrorAndReadingGoesOn
// holds all four together, and
// TestALineAtTheCeilingIsReadAndOneOverItIsNot states the 4 MiB literal
// without reading the constant, as a house-style test does.
//
// A blank line is skipped rather than refused, because a client that ends its
// writes with a newline is not making a mistake: TestABlankLineIsSkipped. A
// last line with no newline of its own is still a line
// (TestALastLineWithoutANewlineIsStillRead), and a carriage return before the
// newline is not part of the message
// (TestACarriageReturnBeforeTheNewlineIsNotPartOfTheMessage).
//
// Output is one compact line per message, every writer through one mutex, so
// the reader and the worker never interleave halves of two answers:
// TestOneCompactLinePerMessage and TestATextWithANewlineInItStaysOneLine.
// HTML escaping is off, so the angle brackets of a label and the ampersands of
// a document go out as themselves: TestATextWithAngleBracketsIsNotEscaped.
//
// # Two halves
//
// The reader parses lines and answers initialize, ping and tools/list on the
// spot, because none of them waits on anything. One worker runs tool calls, one
// at a time, in the order their lines arrived. gapi.Session is not safe for
// concurrent use and the test seams in cmd/gdoc are package variables, so
// serial work is what keeps every command exactly as the terminal has it.
// Answers may leave in any order; the client matches them by id.
//
// The end of the reader ends Serve, which returns once the calls that arrived
// have been answered: TestStdinClosingEndsServe. A call that reached the queue
// was asked for, and the end of stdin is not a reason to pretend it never
// came.
//
// # The errors
//
// An unknown tool is invalid params, -32602, answered in the reader rather
// than behind whatever is running, so a model that guessed a name hears it
// back at once: TestAnUnknownToolIsInvalidParams. Arguments a tool cannot read
// are the tool's own business and come back as a result with isError true, so
// the model can correct itself:
// TestBadArgumentsAreAToolErrorNotAProtocolError. This package validates no
// schema: the schema is for the client's card and the model's own reading, and
// the tool refuses what it does not understand, by name, as every gdoc command
// does.
//
// # The tools
//
// Each text in a Result is one text content item, in order, and an empty
// Result is an empty list rather than null:
// TestEveryTextIsOneContentItemInOrder. The schema goes out as it was written,
// so the order of its properties is the order a card shows them in, and a tool
// with no schema lists the empty object rather than nothing
// (TestAToolWithNoSchemaStillLists). The title goes out twice, at the top
// level and in annotations, so all four protocol versions get the same words,
// and readOnlyHint is the one judgement this package carries about a tool:
// TestToolsListCarriesTitlesSchemasAndHintsInOrder. destructiveHint is always
// false, because nothing gdoc does in chat takes anything away: a reply, a
// comment and a suggestion are each something added.
//
// Add and Remove each tell the client the list changed, and the next
// tools/list agrees: TestAddAndRemoveSendListChanged. A name Remove does not
// find changes nothing and tells nobody, and a name Add already has is
// replaced where it stands, so the order never moves under a client that is
// reading it: TestAddReplacesAToolOfTheSameName. Before Serve there is no
// writer, so a tool added while the server is being built tells nobody and
// simply lists: TestAddBeforeServeTellsNobodyAndStillLists.
//
// # What is not here
//
// No stdin and no stdout. Serve takes a reader and a writer, and only
// cmd/gdoc/main.go names the real ones, which is what lets every test above
// run a whole session against strings in memory. The log is a writer too, and
// never stdout: stdout carries JSON-RPC and nothing else.
//
// TODO(test): the deadline per call, cancellation and the recover around a
// tool are the next task's, and the rules above say nothing about them yet.
package mcp
