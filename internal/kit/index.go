package kit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var headingRx = regexp.MustCompile(`^\s*#{1,3}\s+(.+)$`)

// BuildPlanIndex produces a Markdown bullet list of plans in plansDir,
// sorted by mtime descending, capped at 60. Each plan's first H1/H2 is used
// as the title; falls back to a humanised filename.
func BuildPlanIndex(plansDir string) (string, error) {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	type item struct {
		name  string
		path  string
		mtime time.Time
	}
	var items []item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{
			name:  e.Name(),
			path:  filepath.Join(plansDir, e.Name()),
			mtime: fi.ModTime(),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mtime.After(items[j].mtime) })
	if len(items) > 60 {
		items = items[:60]
	}
	var lines []string
	for _, it := range items {
		title := firstHeading(it.path)
		if title == "" {
			title = humaniseFilename(strings.TrimSuffix(it.name, ".md"))
		}
		date := it.mtime.Format("2006-01-02")
		lines = append(lines, fmt.Sprintf("- [%s](../plans/%s) — %s", title, it.name, date))
	}
	return strings.Join(lines, "\n"), nil
}

// BuildSessionIndex produces a Markdown bullet list of recent Codex sessions
// in sessionsDir.
func BuildSessionIndex(sessionsDir string) (string, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	type item struct {
		name  string
		path  string
		mtime time.Time
	}
	var items []item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".jsonl") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{
			name:  e.Name(),
			path:  filepath.Join(sessionsDir, e.Name()),
			mtime: fi.ModTime(),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mtime.After(items[j].mtime) })
	if len(items) > 40 {
		items = items[:40]
	}
	var lines []string
	for _, it := range items {
		topic := firstSessionTopic(it.path)
		stamp := it.mtime.Format("2006-01-02 15:04")
		lines = append(lines, fmt.Sprintf("- [%s](../sessions/%s) — %s", topic, it.name, stamp))
	}
	return strings.Join(lines, "\n"), nil
}

func firstHeading(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for i := 0; sc.Scan() && i < 30; i++ {
		if m := headingRx.FindStringSubmatch(sc.Text()); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

func firstSessionTopic(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "(unreadable)"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !sc.Scan() {
		return "(empty)"
	}
	var raw map[string]any
	if err := json.Unmarshal(sc.Bytes(), &raw); err != nil {
		return "(unparseable)"
	}
	for _, key := range []string{"prompt", "message", "content", "text"} {
		if v, ok := raw[key].(string); ok && strings.TrimSpace(v) != "" {
			return shorten(v, 80)
		}
	}
	return "(no topic field)"
}

func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func humaniseFilename(s string) string {
	return strings.ReplaceAll(s, "-", " ")
}
