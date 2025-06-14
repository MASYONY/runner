package executors

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// CustomScriptExecutor führt ein beliebiges Skript aus, das im Job definiert ist
func RunCustom(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer) int {
	var cmdStr string
	if script, ok := product["script"]; ok {
		switch v := script.(type) {
		case []interface{}:
			var lines []string
			for _, s := range v {
				switch val := s.(type) {
				case string:
					lines = append(lines, val)
				case []interface{}:
					for _, inner := range val {
						if str, ok := inner.(string); ok {
							lines = append(lines, str)
						}
					}
				}
			}
			cmdStr = strings.Join(lines, "\n")
		case []string:
			cmdStr = strings.Join(v, "\n")
		case string:
			cmdStr = v
		default:
			logWriter.Write([]byte("ERROR: Unbekannter Typ für script: "))
			logWriter.Write([]byte(fmt.Sprintf("%T\n", v)))
		}
	}
	if strings.TrimSpace(cmdStr) == "" {
		logWriter.Write([]byte("ERROR: Kein script im Job definiert\n"))
		return 1
	}
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	for k, v := range variables {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Run(); err != nil {
		logWriter.Write([]byte("ERROR: Custom-Script-Fehler: " + err.Error() + "\n"))
		return 1
	}
	return 0
}
