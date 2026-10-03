// The files one tool call needs on disk, and the directories left behind by a
// process that died.
//
// Where the terminal reads a file, chat sends the value. The CLI's readers are
// strict and they read files, so the server writes the value to a file and
// hands the command a path. The directory is made for the one call and taken
// away after it, whatever the call did, and the path never reaches the person:
// a refusal naming it is rewritten to name the argument the file came from.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gdoc/internal/auth"
)

// mcpTempGlob matches every call directory of every gdoc process, this one's
// and anybody else's. The name carries the process id, so the sweep can tell a
// directory somebody is using from one left by a process that died.
const mcpTempGlob = "gdoc-mcp-*"

// mcpTempPrefix is what this process names its directories, so a directory
// outliving the process says which process made it.
func mcpTempPrefix() string { return fmt.Sprintf("gdoc-mcp-%d-", os.Getpid()) }

// mcpStaleAfter is how long a directory may be in use. It is the longest one
// call can last, spelled here rather than read from internal/mcp: a directory
// older than that belongs to no call, even where the process that made it is
// still running, because that process gave the call up long ago.
const mcpStaleAfter = 200 * time.Second

// callFiles are the files one tool call put on disk and the directory holding
// them. A call that needs no file makes no directory.
type callFiles struct {
	dir string
	// named is a path against the tool argument whose value it holds, so a
	// refusal that names the file can name the argument instead.
	named map[string]string
}

// write puts one value in the call's directory under a fixed name and answers
// the path. The name is never built from an argument: a call that could choose
// its own file name could choose one somewhere else.
func (c *callFiles) write(name, argument, body string) (string, error) {
	if c.dir == "" {
		dir, err := os.MkdirTemp("", mcpTempPrefix())
		if err != nil {
			return "", fmt.Errorf("the directory for this call could not be made: %w", err)
		}
		c.dir = dir
	}
	path := filepath.Join(c.dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", fmt.Errorf("%s could not be written for this call: %w", argument, err)
	}
	if c.named == nil {
		c.named = map[string]string{}
	}
	c.named[path] = argument
	return path, nil
}

// remove takes the directory away. It is deferred, so it runs after an answer,
// after a refusal and after a panic, and it is safe to call twice.
func (c *callFiles) remove() {
	if c.dir == "" {
		return
	}
	os.RemoveAll(c.dir)
	c.dir, c.named = "", nil
}

// name rewrites a sentence so it names the argument rather than the file. The
// commands judge a file the way they judge any file, and their refusals name
// the path they were handed, which in chat is a path nobody typed and nobody
// can look at.
//
// The paths go first and the directory after, because the directory is a prefix
// of every path inside it.
func (c *callFiles) name(text string) string {
	if text == "" {
		return text
	}
	for path, argument := range c.named {
		text = strings.ReplaceAll(text, path, argument)
	}
	if c.dir != "" {
		text = strings.ReplaceAll(text, c.dir, "the directory for this call")
	}
	return text
}

// livePID says whether a process with this id is running. The question is
// auth.ProcessAlive's, which the login lock asks too: one rule about EPERM and
// one about Windows, because a sweep that called a live process dead would take
// that process's own call directory away while it was reading from it.
//
// It is behind a variable so the sweep can be shown a dead process without a
// test having to find a process id nobody is using.
var livePID = auth.ProcessAlive

// sweepCallDirs takes away the call directories nobody is using: the ones whose
// process is gone, and the ones too old to be a call whatever process made
// them. A live process's fresh directory is left alone, because Claude Desktop
// starts two gdoc processes at once, one for chat and one for agent mode
// (docs/v2/MEASURED.md, measurement 3), and one must not sweep the other's
// work out from under it.
//
// Nothing here fails a start. A directory that cannot be read is a directory
// the next sweep sees again.
func sweepCallDirs(root string, now time.Time) {
	matches, err := filepath.Glob(filepath.Join(root, mcpTempGlob))
	if err != nil {
		return
	}
	for _, dir := range matches {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		pid, ok := dirPID(dir)
		if !ok {
			// The name matches the glob and carries no process id, so it is not
			// a directory this code made. Leaving it is the answer that cannot
			// delete somebody else's work.
			continue
		}
		if livePID(pid) && now.Sub(info.ModTime()) < mcpStaleAfter {
			continue
		}
		os.RemoveAll(dir)
	}
}

// dirPID reads the process id out of gdoc-mcp-<pid>-<random>.
func dirPID(dir string) (int, bool) {
	rest := strings.TrimPrefix(filepath.Base(dir), "gdoc-mcp-")
	digits, _, found := strings.Cut(rest, "-")
	if !found {
		return 0, false
	}
	pid, err := strconv.Atoi(digits)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}
