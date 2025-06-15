package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// InterpolateVars ersetzt Platzhalter wie ${jobid.result.data.key} oder ${PREVIOUS_RESULT.key} durch Werte aus jobResults oder result.json.
// jobIDMap: YAML-JobID -> Laufzeit-JobID
// Optional: logger (kann nil sein) für Debug-Ausgaben.
func InterpolateVars(input, workDir string, jobResults map[string]map[string]interface{}, previousJobID string, jobIDMap map[string]string, logger func(string, ...interface{})) string {
	if logger == nil {
		logger = func(string, ...interface{}) {}
	}
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		logger("[InterpolateVars] Platzhalter: %s", match)
		parts := parsePlaceholderPath(key)
		if len(parts) == 0 {
			logger("[InterpolateVars] Ungültiger Platzhalter: %s", match)
			return match
		}
		// PREVIOUS_JOB_ID
		if key == "PREVIOUS_JOB_ID" {
			realPrevID := previousJobID
			if jobIDMap != nil {
				if mapped, ok := jobIDMap[previousJobID]; ok {
					realPrevID = mapped
					logger("[InterpolateVars] Mapping PREVIOUS_JOB_ID '%s' -> '%s'", previousJobID, realPrevID)
				}
			}
			return realPrevID
		}
		// PREVIOUS_RESULT.key
		if parts[0] == "PREVIOUS_RESULT" && len(parts) > 1 && previousJobID != "" {
			if res, ok := jobResults[previousJobID]; ok {
				val := getNested(res, parts[1:])
				if val != nil {
					logger("[InterpolateVars] %s -> %v", match, val)
					return asJSONString(val)
				}
			}
		}
		// jobid.result.data.key...
		jid := parts[0]
		fieldPath := parts[1:]
		// 1. Lookup in jobResults
		if res, ok := jobResults[jid]; ok {
			val := getNested(res, fieldPath)
			if val != nil {
				logger("[InterpolateVars] jobResults[%s] %v -> %v", jid, fieldPath, val)
				return asJSONString(val)
			}
		}
		// 2. Fallback: result.json lesen
		realJobID := jid
		if jobIDMap != nil {
			if mapped, ok := jobIDMap[jid]; ok {
				realJobID = mapped
				logger("[InterpolateVars] Mapping YAML-JobID '%s' -> '%s'", jid, realJobID)
			}
		}
		resultPath := filepath.Join(workDir, realJobID, "result.json")
		if b, err := os.ReadFile(resultPath); err == nil {
			var res map[string]interface{}
			if err := json.Unmarshal(b, &res); err == nil {
				val := getNested(res, fieldPath)
				if val != nil {
					logger("[InterpolateVars] result.json %s %v -> %v", realJobID, fieldPath, val)
					return asJSONString(val)
				}
			}
		}
		logger("[InterpolateVars] Kein Wert für %s gefunden", match)
		return match
	})
}

// parsePlaceholderPath zerlegt einen Platzhalterpfad in Felder, unterstützt Array-Zugriffe (z.B. foo.bar[0].baz)
func parsePlaceholderPath(path string) []string {
	var out []string
	for _, part := range strings.Split(path, ".") {
		for {
			idx := strings.Index(part, "[")
			if idx < 0 {
				out = append(out, part)
				break
			}
			out = append(out, part[:idx])
			end := strings.Index(part, "]")
			if end < 0 || end < idx {
				break
			}
			out = append(out, part[idx:end+1])
			part = part[end+1:]
			if part == "" {
				break
			}
		}
	}
	// Filter leere Felder
	var filtered []string
	for _, s := range out {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// getNested sucht rekursiv nach verschachtelten Feldern, unterstützt Array-Zugriffe wie foo[0]
func getNested(data interface{}, path []string) interface{} {
	cur := data
	for _, p := range path {
		if strings.HasSuffix(p, "]") && strings.Contains(p, "[") {
			// Array-Zugriff
			key := p[:strings.Index(p, "[")]
			idxStr := p[strings.Index(p, "[")+1 : len(p)-1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil || idx < 0 {
				return nil
			}
			if key != "" {
				m, ok := cur.(map[string]interface{})
				if !ok {
					return nil
				}
				arr, ok := m[key].([]interface{})
				if !ok || idx >= len(arr) {
					return nil
				}
				cur = arr[idx]
			} else {
				arr, ok := cur.([]interface{})
				if !ok || idx >= len(arr) {
					return nil
				}
				cur = arr[idx]
			}
		} else {
			m, ok := cur.(map[string]interface{})
			if !ok {
				return nil
			}
			v, ok := m[p]
			if !ok {
				return nil
			}
			cur = v
		}
	}
	return cur
}

// asJSONString gibt einen Wert als String zurück, Strings bleiben unverändert, andere Typen werden als JSON serialisiert.
func asJSONString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
