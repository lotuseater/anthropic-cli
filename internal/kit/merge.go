package kit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// stripWizardKitMeta removes top-level _wizard_kit and *_wizard_kit_* keys
// from a parsed JSON tree before merging. Those keys document the template
// for humans and must not pollute the user's settings.json.
func stripWizardKitMeta(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if k == "_wizard_kit" || strings.HasPrefix(k, "_wizard_kit_") {
			continue
		}
		if sub, ok := v.(map[string]any); ok {
			out[k] = stripWizardKitMeta(sub)
		} else if arr, ok := v.([]any); ok {
			out[k] = stripArrayMeta(arr)
		} else {
			out[k] = v
		}
	}
	return out
}

func stripArrayMeta(arr []any) []any {
	out := make([]any, 0, len(arr))
	for _, v := range arr {
		if sub, ok := v.(map[string]any); ok {
			out = append(out, stripWizardKitMeta(sub))
		} else {
			out = append(out, v)
		}
	}
	return out
}

// mergeJSON deep-merges delta into existing. Maps merge recursively. Arrays
// of objects with a "wizard_kit_id" key are matched and replaced by id;
// other arrays are unioned by JSON-equality.
func mergeJSON(existing, delta map[string]any) map[string]any {
	if existing == nil {
		existing = map[string]any{}
	}
	for k, dv := range delta {
		ev, ok := existing[k]
		if !ok {
			existing[k] = dv
			continue
		}
		switch dvTyped := dv.(type) {
		case map[string]any:
			if evMap, ok := ev.(map[string]any); ok {
				existing[k] = mergeJSON(evMap, dvTyped)
				continue
			}
			existing[k] = dvTyped
		case []any:
			if evArr, ok := ev.([]any); ok {
				existing[k] = mergeArrayUnion(evArr, dvTyped)
				continue
			}
			existing[k] = dvTyped
		default:
			existing[k] = dv
		}
	}
	return existing
}

func mergeArrayUnion(existing, delta []any) []any {
	out := make([]any, len(existing))
	copy(out, existing)

	for _, d := range delta {
		dMap, dIsMap := d.(map[string]any)
		if dIsMap {
			if id, ok := dMap["wizard_kit_id"].(string); ok && id != "" {
				replaced := false
				for i, e := range out {
					eMap, ok := e.(map[string]any)
					if !ok {
						continue
					}
					if eid, ok := eMap["wizard_kit_id"].(string); ok && eid == id {
						out[i] = d
						replaced = true
						break
					}
				}
				if !replaced {
					out = append(out, d)
				}
				continue
			}
		}
		// Plain array: dedupe by canonical JSON value.
		if !arrayContainsJSON(out, d) {
			out = append(out, d)
		}
	}
	return out
}

func arrayContainsJSON(arr []any, target any) bool {
	tBytes, err := json.Marshal(target)
	if err != nil {
		return false
	}
	for _, e := range arr {
		eBytes, err := json.Marshal(e)
		if err != nil {
			continue
		}
		if bytes.Equal(tBytes, eBytes) {
			return true
		}
	}
	return false
}

// mergeJSONFile loads existing target JSON, merges delta, writes back. If
// dryRun is true, no write occurs but the merged output is returned.
func mergeJSONFile(target string, delta map[string]any, dryRun bool) (merged map[string]any, err error) {
	delta = stripWizardKitMeta(delta)
	existing := map[string]any{}
	if data, readErr := os.ReadFile(target); readErr == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &existing); err != nil {
			return nil, fmt.Errorf("parse %s: %w", target, err)
		}
	}
	merged = mergeJSON(existing, delta)
	if dryRun {
		return merged, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, err
	}
	return merged, os.WriteFile(target, out, 0o644)
}

// mergeMarkdownSection replaces or appends a delimited block in a Markdown
// file. Delimiters are `<!-- <marker> BEGIN ... -->` and `<!-- <marker> END -->`.
// Content outside the markers is preserved.
func mergeMarkdownSection(target, marker, body string, dryRun bool) error {
	beginPat := regexp.MustCompile(`(?ms)<!--\s+` + regexp.QuoteMeta(marker) + `\s+BEGIN.*?<!--\s+` + regexp.QuoteMeta(marker) + `\s+END\s+-->`)
	existing := ""
	if data, err := os.ReadFile(target); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}
	stripped := strings.TrimRight(beginPat.ReplaceAllString(existing, ""), "\r\n")
	newBlock := strings.TrimRight(body, "\r\n")
	var merged string
	if strings.TrimSpace(stripped) == "" {
		merged = newBlock + "\n"
	} else {
		merged = stripped + "\n\n" + newBlock + "\n"
	}
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(merged), 0o644)
}

// mergeTomlSection appends or replaces a `# === wizard-kit:codex BEGIN ===` /
// END block in a TOML file. We don't parse TOML — we just delimit and replace.
func mergeTomlSection(target, body string, dryRun bool) error {
	begin := "# === wizard-kit:codex BEGIN ==="
	end := "# === wizard-kit:codex END ==="
	rx := regexp.MustCompile(`(?ms)` + regexp.QuoteMeta(begin) + `.*?` + regexp.QuoteMeta(end))
	existing := ""
	if data, err := os.ReadFile(target); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}
	stripped := strings.TrimRight(rx.ReplaceAllString(existing, ""), "\r\n")
	block := begin + "\n" + strings.TrimRight(body, "\r\n") + "\n" + end
	var merged string
	if strings.TrimSpace(stripped) == "" {
		merged = block + "\n"
	} else {
		merged = stripped + "\n\n" + block + "\n"
	}
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(merged), 0o644)
}

// updateIndexBlock replaces a `<!-- <marker> BEGIN ... --> ... <!-- <marker> END -->`
// block in a Markdown file with new index body. Creates the block at end if
// missing.
func updateIndexBlock(target, marker, body string, dryRun bool) error {
	rx := regexp.MustCompile(`(?ms)<!--\s+` + regexp.QuoteMeta(marker) + `\s+BEGIN.*?<!--\s+` + regexp.QuoteMeta(marker) + `\s+END\s+-->`)
	existing := ""
	if data, err := os.ReadFile(target); err == nil {
		existing = string(data)
	} else {
		// no target — silently skip (caller should ensure file exists when meaningful).
		return nil
	}
	newBlock := fmt.Sprintf("<!-- %s BEGIN -->\n%s\n<!-- %s END -->", marker, strings.TrimRight(body, "\r\n"), marker)
	var merged string
	if rx.MatchString(existing) {
		merged = rx.ReplaceAllString(existing, newBlock)
	} else {
		merged = strings.TrimRight(existing, "\r\n") + "\n\n" + newBlock + "\n"
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(target, []byte(merged), 0o644)
}
