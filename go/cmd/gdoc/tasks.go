// What a publish does with the task lines a note held, before anything leaves
// the machine.
//
// internal/body reads each task apart and leaves the owner as the note wrote
// it, "@anna" or "@anna@example.com". A chip in the document names a whole
// address, so a bare name has to be finished against the domain in effect, and
// a run that cannot finish one must send nothing at all: a publish that
// uploaded first and failed here would leave a document whose tasks are
// characters, and the author would publish again to fix a note that is already
// right. So this happens after the render and before the session is opened.
//
// `build` calls none of it. The docx keeps the token as the note wrote it,
// nothing is sent to anybody, and there is nothing to resolve.

package main

import (
	"fmt"

	"gdoc/internal/body"
	"gdoc/internal/publish"
	"gdoc/internal/tasks"
)

// resolveTasks finishes every owner against the domain in effect, and answers
// the tasks a publish sends. It refuses by naming the token it could not
// finish and the setting that would finish it, which is internal/tasks's own
// sentence: the note is the only place the author can see what they wrote.
// TestABareOwnerWithNoDomainIsRefusedBeforeTheUpload.
func resolveTasks(list []body.Task, domain string) ([]publish.Task, error) {
	if len(list) == 0 {
		return nil, nil
	}
	out := make([]publish.Task, 0, len(list))
	for _, t := range list {
		owner, err := resolveOwner(t.Owner, domain)
		if err != nil {
			return nil, fmt.Errorf("the task %q names %s: %w", t.Words, t.Owner, err)
		}
		out = append(out, publish.Task{
			Words:  t.Words,
			Owner:  owner,
			Due:    t.Due,
			Done:   t.Done,
			Tokens: t.Tokens,
		})
	}
	return out, nil
}

// resolveOwner is one owner token as a whole address, or nothing at all where
// the item names nobody.
func resolveOwner(token, domain string) (string, error) {
	if token == "" {
		return "", nil
	}
	owner, err := tasks.ReadOwner(token)
	if err != nil {
		return "", err
	}
	return owner.Resolve(domain)
}
