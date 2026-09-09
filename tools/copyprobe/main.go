// copyprobe answers one question, and then it should be deleted.
//
// Does `files.copy?copyComments=true` actually carry a Google Doc's anchored
// comments and its pending suggestions into the copy?
//
// Why it is being asked. gdoc's docs/v2/BLOCKED-BY-API.md records, measured on
// 2026-08-29, that `files.copy` drops all comments while keeping all anchors.
// That fact is load-bearing: the in-place restyle exists because replacing a
// body destroys comment anchors, and its acceptance test has to restyle a copy
// of a document holding a real anchored comment. If copies cannot carry
// comments, the copy has to be made by hand in a browser.
//
// On 2026-09-04 Google documented a `copyComments` parameter, five days after
// that measurement. So the recorded fact may be stale rather than wrong.
//
// This program does not trust the documentation, and that is deliberate rather
// than rude. In this project the reference has been wrong about exactly this
// kind of thing before: `writeMode` is absent from the public discovery
// document altogether, and DECISIONS.md records four 200 responses that lied in
// a single day. So the parameter is measured against a real document.
//
// What it does, all inside the Drive test folder, cleaning up after itself:
//
//  1. Creates a source document.
//  2. Writes a sentence into it.
//  3. Anchors a comment to specific words, through the Docs API, which is the
//     one route that makes a real anchor.
//  4. Leaves a pending suggestion in it, by writing in SUGGEST mode.
//  5. Copies it twice: once with copyComments=true, once without, because a
//     control is what turns a result into a measurement.
//  6. Asks of each copy: are the comments there, is the comment still anchored,
//     is the suggestion still pending?
//  7. Trashes all three documents.
//
// The anchor question is answered from the docx export, not from
// `comments.list`. That is the trap this whole area turns on: Drive keeps
// returning a comment's original `anchor` and `quotedFileContent` after the
// anchor has been destroyed, so the listing reports a broken anchor as healthy.
// The export carries commentRangeStart and commentRangeEnd markers, and those
// only exist where text is genuinely enclosed.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	testFolder = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"

	// The sentence, and the words the comment is anchored to. They occur once.
	sentence = "The supplier register is reviewed annually by the operations team.\n"
	anchored = "annually"
	suggest  = " and audited"
)

var (
	keep = flag.Bool("keep", false, "do not trash the documents, so they can be opened and looked at")
	// trashID exists because the first version of this program left a document
	// behind: it called os.Exit on an error, and os.Exit does not run deferred
	// functions. The structure is fixed, and this stays for the run that dies
	// some other way.
	trashID = flag.String("trash", "", "trash this document id and exit, for cleaning up after a run that died")
)

type client struct {
	bearer string
	http   *http.Client
}

func main() {
	folder := flag.String("folder", testFolder, "the Drive folder to create in")
	flag.Parse()

	if *trashID != "" {
		tok, err := accessToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "copyprobe: %v\n", err)
			os.Exit(1)
		}
		c := &client{bearer: tok, http: &http.Client{Timeout: 60 * time.Second}}
		if err := c.trash(*trashID); err != nil {
			fmt.Fprintf(os.Stderr, "copyprobe: %s could not be trashed: %v\n", *trashID, err)
			os.Exit(1)
		}
		fmt.Printf("trashed %s\n", *trashID)
		return
	}
	if err := run(*folder); err != nil {
		fmt.Fprintf(os.Stderr, "copyprobe: %v\n", err)
		os.Exit(1)
	}
}

// run does the work and returns, so main can exit on an error without skipping
// the cleanup. os.Exit does not run deferred functions, and the first version of
// this program left a document in the folder proving it.
func run(folder string) error {
	tok, err := accessToken()
	if err != nil {
		return err
	}
	c := &client{bearer: tok, http: &http.Client{Timeout: 60 * time.Second}}

	fmt.Printf("copyprobe, %s\n", time.Now().Format(time.RFC1123))
	fmt.Printf("folder %s\n\n", folder)

	var made []string
	defer func() {
		if *keep {
			fmt.Println("\n-keep was set, so these are still in the folder:")
			for _, id := range made {
				fmt.Printf("  https://docs.google.com/document/d/%s/edit\n", id)
			}
			return
		}
		fmt.Println()
		for _, id := range made {
			if err := c.trash(id); err != nil {
				fmt.Printf("  NOT TRASHED %s: %v\n", id, err)
			}
		}
		fmt.Printf("trashed %d document(s)\n", len(made))
	}()

	src, err := c.buildSource(folder)
	if err != nil {
		// The id comes back even when a later step failed, so the document is
		// recorded for trashing before the error is returned.
		if src != "" {
			made = append(made, src)
		}
		return err
	}
	made = append(made, src)
	fmt.Printf("source document %s\n", src)

	before := c.inspect(src)
	fmt.Printf("  source holds: %s\n\n", before)
	if !before.comments || !before.anchoredInExport || !before.suggestions {
		return fmt.Errorf("the source document does not hold what the measurement needs, so no copy was made and nothing below would have meant anything")
	}

	type run struct {
		name  string
		param string
	}
	results := map[string]state{}
	for _, r := range []run{
		{"copy WITH copyComments=true", "&copyComments=true"},
		{"copy WITHOUT the parameter", ""},
	} {
		id, err := c.copy(src, folder, r.param)
		if err != nil {
			fmt.Printf("%-30s FAILED: %v\n", r.name, err)
			continue
		}
		made = append(made, id)
		st := c.inspect(id)
		results[r.name] = st
		fmt.Printf("%-30s %s\n", r.name, st)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 78))
	with, okWith := results["copy WITH copyComments=true"]
	without, okWithout := results["copy WITHOUT the parameter"]
	switch {
	case !okWith:
		fmt.Println("VERDICT: the copy with copyComments=true could not be made. Read the error above.")
	case with.comments && with.anchoredInExport:
		fmt.Println("VERDICT: copyComments=true carries the comment, and it is still anchored.")
		fmt.Println("         BLOCKED-BY-API.md's \"files.copy drops all comments\" is stale and")
		fmt.Println("         should be corrected with today's date rather than deleted.")
		if okWithout && !without.comments {
			fmt.Println("         The control behaved as recorded, so the old measurement was")
			fmt.Println("         right about the default and the parameter is genuinely new.")
		}
		if !with.suggestions {
			fmt.Println("         Note: the pending suggestion did NOT survive, whatever the")
			fmt.Println("         guide says about suggestions. That is a second fact worth")
			fmt.Println("         recording, and it constrains the ten-feature run.")
		}
	case with.comments && !with.anchoredInExport:
		fmt.Println("VERDICT: the comment came across and it is NOT anchored.")
		fmt.Println("         This is the case comments.list would have reported as healthy,")
		fmt.Println("         which is why the export is the witness. An unanchored copy is")
		fmt.Println("         no use for an acceptance test about anchors.")
	default:
		fmt.Println("VERDICT: copyComments=true did not carry the comment.")
		fmt.Println("         The recorded behaviour stands, and the acceptance copy cannot be")
		fmt.Println("         made through the API.")
	}
	return nil
}

// state is what a document holds, in the three terms the question is asked in.
type state struct {
	comments         bool
	anchoredInExport bool
	suggestions      bool
	detail           string
}

func (s state) String() string {
	f := func(b bool) string {
		if b {
			return "yes"
		}
		return "NO"
	}
	return fmt.Sprintf("comments=%-3s anchored=%-3s pending suggestion=%-3s %s",
		f(s.comments), f(s.anchoredInExport), f(s.suggestions), s.detail)
}

func (c *client) inspect(id string) state {
	var st state
	var list struct {
		Comments []struct {
			Content           string `json:"content"`
			QuotedFileContent struct {
				Value string `json:"value"`
			} `json:"quotedFileContent"`
		} `json:"comments"`
	}
	if err := c.getJSON("https://www.googleapis.com/drive/v3/files/"+id+
		"/comments?fields=comments(content,quotedFileContent)&supportsAllDrives=true", &list); err != nil {
		st.detail = "comments.list failed: " + oneLine(err.Error())
	} else {
		st.comments = len(list.Comments) > 0
	}

	// The export is the honest witness. comments.list keeps reporting an anchor
	// that no longer holds any text, so it cannot answer this.
	if b, err := c.exportDocx(id); err != nil {
		st.detail += " export failed: " + oneLine(err.Error())
	} else {
		st.anchoredInExport = docxHasAnchor(b)
	}

	var doc struct {
		Body struct {
			Content []struct {
				Paragraph struct {
					Elements []struct {
						TextRun struct {
							SuggestedInsertionIds []string `json:"suggestedInsertionIds"`
						} `json:"textRun"`
					} `json:"elements"`
				} `json:"paragraph"`
			} `json:"content"`
		} `json:"body"`
	}
	if err := c.getJSON("https://docs.googleapis.com/v1/documents/"+id+
		"?suggestionsViewMode=SUGGESTIONS_INLINE", &doc); err == nil {
		for _, e := range doc.Body.Content {
			for _, el := range e.Paragraph.Elements {
				if len(el.TextRun.SuggestedInsertionIds) > 0 {
					st.suggestions = true
				}
			}
		}
	}
	return st
}

// docxHasAnchor reports whether the export carries a comment range around some
// text. commentRangeStart exists only where a comment genuinely encloses
// content, which is the fact Drive's own listing cannot be asked for.
func docxHasAnchor(b []byte) bool {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return false
	}
	for _, f := range z.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return false
		}
		defer rc.Close()
		body, err := io.ReadAll(rc)
		if err != nil {
			return false
		}
		return bytes.Contains(body, []byte("commentRangeStart"))
	}
	return false
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 100 {
		s = s[:100] + "..."
	}
	return s
}

var _ = json.Marshal
