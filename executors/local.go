package executors

import (
	"io"
	"os/exec"
	"strings"
)

// LocalExecutor führt die Befehle direkt auf dem Host aus
func RunLocal(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer) int {
	var cmdStr string
	if commands, ok := product["commands"]; ok {
		switch v := commands.(type) {
		case []interface{}:
			var lines []string
			for _, s := range v {
				if str, ok := s.(string); ok {
					lines = append(lines, str)
				}
			}
			cmdStr = strings.Join(lines, "\n")
		case string:
			cmdStr = v
		}
	}
	if strings.TrimSpace(cmdStr) == "" {
		logWriter.Write([]byte("ERROR: Keine commands im Job definiert\n"))
		return 1
	}
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	for k, v := range variables {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Run(); err != nil {
		logWriter.Write([]byte("ERROR: Local-Executor-Fehler: " + err.Error() + "\n"))
		return 1
	}
	return 0
}
