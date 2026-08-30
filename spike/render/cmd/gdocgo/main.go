// Command gdocgo is the throwaway Go renderer, built to measure how close a Go
// port can get to the Python one's output. It is not a replacement for gdoc: it
// does the render and the publish, and nothing else.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"spike/gdocgo/internal/drive"
	"spike/gdocgo/internal/generate"
	"spike/gdocgo/internal/render"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = cmdBuild(os.Args[2:])
	case "generate":
		err = cmdGenerate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gdocgo:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `gdocgo build    <markdown> --template <docx> [--out <docx>] [--links]
gdocgo generate <markdown> --template <docx> --folder <id> [--out <docx>] [--links]
`)
}

// flagsFirst moves positional arguments behind the flags. Go's flag package
// stops at the first non-flag word, so `build note.md --template x` would
// otherwise parse no flags at all and fail naming the template.
func flagsFirst(args []string) []string {
	var flagArgs, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}
		flagArgs = append(flagArgs, arg)
		// A flag written as two words takes the next one with it.
		if !strings.Contains(arg, "=") && i+1 < len(args) &&
			!strings.HasPrefix(args[i+1], "-") && !boolFlags[strings.TrimLeft(arg, "-")] {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}
	return append(flagArgs, positional...)
}

var boolFlags = map[string]bool{"links": true}

func defaultToken() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gdoc-agent", "oauth-token.json")
}

func cmdBuild(args []string) error {
	flags := flag.NewFlagSet("build", flag.ExitOnError)
	args = flagsFirst(args)
	template := flags.String("template", "", "path to the master .docx")
	out := flags.String("out", "", "where to write the .docx")
	title := flags.String("title", "", "override the note's own title for this run")
	links := flags.Bool("links", false, "write real hyperlinks rather than link text alone")
	pagesFile := flags.String("pages", "", "JSON map of heading text to page number")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() < 1 {
		return fmt.Errorf("which markdown file?")
	}

	pages, err := readPages(*pagesFile)
	if err != nil {
		return err
	}
	result, err := render.Build(flags.Arg(0), *out, render.Options{
		Template: *template, Title: *title, Pages: pages, Hyperlinks: *links,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"docx":     result.DocxPath,
		"title":    result.Title,
		"blocks":   result.Blocks,
		"entries":  result.Entries,
	})
}

func readPages(path string) (map[string]int, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pages := map[string]int{}
	return pages, json.Unmarshal(data, &pages)
}

func cmdGenerate(args []string) error {
	flags := flag.NewFlagSet("generate", flag.ExitOnError)
	args = flagsFirst(args)
	template := flags.String("template", "", "path to the master .docx")
	out := flags.String("out", "", "where to write the .docx")
	folder := flags.String("folder", "", "Drive folder id to publish into")
	name := flags.String("name", "", "the document's name in Drive")
	title := flags.String("title", "", "override the note's own title for this run")
	links := flags.Bool("links", false, "write real hyperlinks rather than link text alone")
	token := flags.String("token", defaultToken(), "path to the OAuth token")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() < 1 {
		return fmt.Errorf("which markdown file?")
	}
	if *folder == "" {
		return fmt.Errorf("--folder is required: nothing here guesses where to publish")
	}
	outPath := *out
	if outPath == "" {
		outPath = filepath.Join(os.TempDir(), "gdocgo-out.docx")
	}
	documentName := *name
	if documentName == "" {
		documentName = "gdocgo " + filepath.Base(flags.Arg(0))
	}

	ctx := context.Background()
	service, err := drive.New(ctx, *token, []string{*folder})
	if err != nil {
		return err
	}
	result, err := generate.Generate(ctx, service, flags.Arg(0), documentName, outPath,
		*folder, render.Options{Template: *template, Title: *title, Hyperlinks: *links})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"docx":   result.DocxPath,
		"doc_id": result.DocID,
		"link":   result.Link,
		"reason": result.Reason,
		"drift":  result.Drift,
	})
}
