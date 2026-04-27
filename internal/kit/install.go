package kit

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options controls Install/Rollback behaviour.
type Options struct {
	ClaudeOnly bool
	CodexOnly  bool
	DryRun     bool
	Out        io.Writer // nil → os.Stdout
}

// Install applies the kit to the configured targets.
func Install(home string, opts Options) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	step(out, "Wizard Kit installer (Go) v%s", Version)
	if opts.DryRun {
		note(out, "mode: DRY-RUN (no writes)")
	} else {
		note(out, "mode: INSTALL")
	}
	if !opts.CodexOnly {
		if err := installClaude(home, opts, out); err != nil {
			return fmt.Errorf("claude install: %w", err)
		}
	}
	if !opts.ClaudeOnly {
		if err := installCodex(home, opts, out); err != nil {
			return fmt.Errorf("codex install: %w", err)
		}
	}
	done(out, "install complete")
	return nil
}

func installClaude(home string, opts Options, out io.Writer) error {
	target := filepath.Join(home, ".claude")
	step(out, "Installing Claude Code kit -> %s", target)
	backup, err := newBackupDir(target, opts.DryRun)
	if err != nil {
		return err
	}

	// 1. settings.json merge.
	settingsTarget := filepath.Join(target, "settings.json")
	deltaBytes, err := Payload.ReadFile("payload/templates/claude-code/settings.json.template")
	if err != nil {
		return err
	}
	var delta map[string]any
	if err := json.Unmarshal(deltaBytes, &delta); err != nil {
		return fmt.Errorf("parse settings template: %w", err)
	}
	if _, err := backupFile(settingsTarget, backup, target, opts.DryRun); err != nil {
		return err
	}
	if _, err := mergeJSONFile(settingsTarget, delta, opts.DryRun); err != nil {
		return fmt.Errorf("merge settings.json: %w", err)
	}
	change(out, "merge settings.json -> %s", settingsTarget)

	// 2. CLAUDE.md.
	claudeMd := filepath.Join(target, "CLAUDE.md")
	if _, err := backupFile(claudeMd, backup, target, opts.DryRun); err != nil {
		return err
	}
	bodyBytes, err := Payload.ReadFile("payload/templates/claude-code/CLAUDE.md.template")
	if err != nil {
		return err
	}
	if err := mergeMarkdownSection(claudeMd, "wizard-kit:claude-md", string(bodyBytes), opts.DryRun); err != nil {
		return err
	}
	change(out, "merge CLAUDE.md -> %s", claudeMd)

	// 3. memory/MEMORY.md (create if missing, then update plan-index block).
	memoryDir := filepath.Join(target, "memory")
	memoryMd := filepath.Join(memoryDir, "MEMORY.md")
	if !opts.DryRun {
		if err := os.MkdirAll(memoryDir, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(memoryMd); os.IsNotExist(err) {
		tplBytes, err := Payload.ReadFile("payload/templates/claude-code/MEMORY.md.template")
		if err != nil {
			return err
		}
		if !opts.DryRun {
			if err := os.WriteFile(memoryMd, tplBytes, 0o644); err != nil {
				return err
			}
		}
		change(out, "create %s", memoryMd)
	} else {
		if _, err := backupFile(memoryMd, backup, target, opts.DryRun); err != nil {
			return err
		}
	}
	planIndex, err := BuildPlanIndex(filepath.Join(target, "plans"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(planIndex) == "" {
		planIndex = "<!-- (no plans found in ~/.claude/plans/) -->"
	}
	if err := updateIndexBlock(memoryMd, "wizard-kit:plan-index", planIndex, opts.DryRun); err != nil {
		return err
	}
	change(out, "rebuild plan-index in %s", memoryMd)

	// 4. Skills.
	if err := copyEmbeddedSkills("payload/skills/claude-code", filepath.Join(target, "skills"), backup, target, opts, out); err != nil {
		return err
	}

	// 5. Per-project locals.
	if err := dropPerProjectLocals(home, opts, out); err != nil {
		return err
	}

	// 6. Manifest.
	return updateManifest(target, manifestEntry{
		Version:   Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "install",
		Installer: "ant kit install",
		BackupDir: relPathOrEmpty(target, backup),
	}, opts.DryRun)
}

func installCodex(home string, opts Options, out io.Writer) error {
	target := filepath.Join(home, ".codex")
	step(out, "Installing Codex kit -> %s", target)
	backup, err := newBackupDir(target, opts.DryRun)
	if err != nil {
		return err
	}

	// 1. config.toml — replace marker block.
	configToml := filepath.Join(target, "config.toml")
	if _, err := backupFile(configToml, backup, target, opts.DryRun); err != nil {
		return err
	}
	tomlBytes, err := Payload.ReadFile("payload/templates/codex/config.toml.template")
	if err != nil {
		return err
	}
	if err := mergeTomlSection(configToml, string(tomlBytes), opts.DryRun); err != nil {
		return err
	}
	change(out, "merge config.toml -> %s", configToml)

	// 2. AGENTS.md.
	agentsMd := filepath.Join(target, "AGENTS.md")
	if _, err := backupFile(agentsMd, backup, target, opts.DryRun); err != nil {
		return err
	}
	bodyBytes, err := Payload.ReadFile("payload/templates/codex/AGENTS.md.template")
	if err != nil {
		return err
	}
	if err := mergeMarkdownSection(agentsMd, "wizard-kit:agents-md", string(bodyBytes), opts.DryRun); err != nil {
		return err
	}
	change(out, "merge AGENTS.md -> %s", agentsMd)

	// 3. Session index.
	sessIndex, err := BuildSessionIndex(filepath.Join(target, "sessions"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(sessIndex) == "" {
		sessIndex = "<!-- (no sessions found in ~/.codex/sessions/) -->"
	}
	if err := updateIndexBlock(agentsMd, "wizard-kit:session-index", sessIndex, opts.DryRun); err != nil {
		return err
	}
	change(out, "rebuild session-index in %s", agentsMd)

	// 4. Skills.
	if err := copyEmbeddedSkills("payload/skills/codex", filepath.Join(target, "skills"), backup, target, opts, out); err != nil {
		return err
	}

	return updateManifest(target, manifestEntry{
		Version:   Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "install",
		Installer: "ant kit install",
		BackupDir: relPathOrEmpty(target, backup),
	}, opts.DryRun)
}

func copyEmbeddedSkills(srcRoot, destRoot, backup, target string, opts Options, out io.Writer) error {
	if !opts.DryRun {
		if err := os.MkdirAll(destRoot, 0o755); err != nil {
			return err
		}
	}
	return fs.WalkDir(Payload, srcRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(p) != "SKILL.md" {
			return nil
		}
		// p looks like payload/skills/claude-code/<skill>/SKILL.md
		rel, err := filepath.Rel(srcRoot, p)
		if err != nil {
			return err
		}
		destFile := filepath.Join(destRoot, filepath.ToSlash(rel))
		// Backup existing.
		if _, err := backupFile(destFile, backup, target, opts.DryRun); err != nil {
			return err
		}
		data, err := Payload.ReadFile(p)
		if err != nil {
			return err
		}
		if !opts.DryRun {
			if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(destFile, data, 0o644); err != nil {
				return err
			}
		}
		change(out, "skill %s", strings.TrimSuffix(rel, "/SKILL.md"))
		return nil
	})
}

func dropPerProjectLocals(home string, opts Options, out io.Writer) error {
	githubRoot := filepath.Join(home, "Documents", "GitHub")
	if _, err := os.Stat(githubRoot); err != nil {
		warn(out, "skipping per-project local: no GitHub root at %s", githubRoot)
		return nil
	}
	type detector struct {
		Family string
		Marker string
	}
	detectors := []detector{
		{"php", "composer.json"},
		{"cpp", "CMakeLists.txt"},
		{"python", "pyproject.toml"},
	}
	entries, err := os.ReadDir(githubRoot)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		proj := filepath.Join(githubRoot, e.Name())
		for _, det := range detectors {
			marker := filepath.Join(proj, det.Marker)
			if _, err := os.Stat(marker); err != nil {
				continue
			}
			localDest := filepath.Join(proj, ".claude", "settings.local.json")
			if _, err := os.Stat(localDest); err == nil {
				note(out, "skip %s (settings.local.json exists)", proj)
				break
			}
			srcPath := fmt.Sprintf("payload/templates/claude-code/per-project/%s/.claude/settings.local.json", det.Family)
			data, err := Payload.ReadFile(srcPath)
			if err != nil {
				warn(out, "per-project template missing: %s", srcPath)
				break
			}
			if !opts.DryRun {
				if err := os.MkdirAll(filepath.Dir(localDest), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(localDest, data, 0o644); err != nil {
					return err
				}
			}
			change(out, "per-project [%s] -> %s", det.Family, localDest)
			break
		}
	}
	return nil
}

// ---- backup / manifest helpers ---------------------------------------------

func newBackupDir(target string, dryRun bool) (string, error) {
	stamp := time.Now().UTC().Format("20060102-150405")
	dir := filepath.Join(target, ".wizard-kit-backup", stamp)
	if !dryRun {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func backupFile(src, backup, target string, dryRun bool) (bool, error) {
	if _, err := os.Stat(src); err != nil {
		return false, nil
	}
	rel, err := filepath.Rel(target, src)
	if err != nil {
		return false, err
	}
	dest := filepath.Join(backup, rel)
	if dryRun {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return false, err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(dest, data, 0o644)
}

func relPathOrEmpty(base, p string) string {
	rel, err := filepath.Rel(base, p)
	if err != nil {
		return p
	}
	return rel
}

type manifestEntry struct {
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Installer string `json:"installer"`
	BackupDir string `json:"backup_dir,omitempty"`
	From      string `json:"from,omitempty"`
}

type manifest struct {
	History []manifestEntry `json:"history"`
	Last    *manifestEntry  `json:"last,omitempty"`
}

func updateManifest(target string, entry manifestEntry, dryRun bool) error {
	if dryRun {
		return nil
	}
	path := filepath.Join(target, ".wizard-kit-manifest.json")
	var m manifest
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	m.History = append(m.History, entry)
	cur := entry
	m.Last = &cur
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// ---- output helpers --------------------------------------------------------

func step(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "==> "+format+"\n", args...)
}
func note(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "    "+format+"\n", args...)
}
func change(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "    + "+format+"\n", args...)
}
func warn(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "    ! "+format+"\n", args...)
}
func done(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "\n"+format+"\n", args...)
}
