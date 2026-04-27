package kit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Rollback restores the most recent backup snapshot under each enabled
// target's .wizard-kit-backup/ folder.
func Rollback(home string, opts Options) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	step(out, "Wizard Kit rollback (Go) v%s", Version)
	if !opts.CodexOnly {
		if err := rollbackTarget(filepath.Join(home, ".claude"), opts, out); err != nil {
			return fmt.Errorf("claude rollback: %w", err)
		}
	}
	if !opts.ClaudeOnly {
		if err := rollbackTarget(filepath.Join(home, ".codex"), opts, out); err != nil {
			return fmt.Errorf("codex rollback: %w", err)
		}
	}
	done(out, "rollback complete")
	return nil
}

func rollbackTarget(target string, opts Options, out io.Writer) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		warn(out, "target missing: %s", target)
		return nil
	}
	step(out, "Rolling back -> %s", target)

	backupRoot := filepath.Join(target, ".wizard-kit-backup")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		warn(out, "no backups to restore: %s", backupRoot)
		return nil
	}
	var snaps []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			snaps = append(snaps, e)
		}
	}
	if len(snaps) == 0 {
		warn(out, "no backup snapshots present")
		return nil
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].Name() > snaps[j].Name() })
	latest := snaps[0]
	latestPath := filepath.Join(backupRoot, latest.Name())
	note(out, "restoring from %s", latestPath)

	err = filepath.Walk(latestPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(latestPath, p)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if opts.DryRun {
			note(out, "would restore %s", rel)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
		change(out, "restore %s", rel)
		return nil
	})
	if err != nil {
		return err
	}
	return updateManifest(target, manifestEntry{
		Version:   Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "rollback",
		Installer: "ant kit rollback",
		From:      latest.Name(),
	}, opts.DryRun)
}
