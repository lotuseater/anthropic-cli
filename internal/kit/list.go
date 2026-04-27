package kit

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// ListAssets prints a tree of skills + templates from the embedded payload.
func ListAssets(out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintf(out, "Wizard Kit v%s — embedded payload\n\n", Version)
	if err := listGroup(out, "Claude Code skills", "payload/skills/claude-code"); err != nil {
		return err
	}
	if err := listGroup(out, "Codex skills", "payload/skills/codex"); err != nil {
		return err
	}
	if err := listGroup(out, "Claude Code templates", "payload/templates/claude-code"); err != nil {
		return err
	}
	if err := listGroup(out, "Codex templates", "payload/templates/codex"); err != nil {
		return err
	}
	return nil
}

func listGroup(out io.Writer, title, root string) error {
	fmt.Fprintf(out, "## %s\n", title)
	var paths []string
	err := fs.WalkDir(Payload, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, strings.TrimPrefix(p, root+"/"))
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, p := range paths {
		fmt.Fprintf(out, "  - %s\n", p)
	}
	fmt.Fprintln(out)
	return nil
}
