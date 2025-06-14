package executors

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// SSHExecutor führt die Befehle per SSH auf einem entfernten Host aus
func RunSSH(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer) int {
	host, ok := product["host"].(string)
	if !ok || host == "" {
		logWriter.Write([]byte("ERROR: Kein SSH-Host im Job definiert\n"))
		return 1
	}
	user := "root"
	if u, ok := product["user"].(string); ok && u != "" {
		user = u
	}
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
	sshCmd := fmt.Sprintf("ssh %s@%s '%s'", user, host, strings.ReplaceAll(cmdStr, "'", "'\\''"))
	cmd := exec.Command("sh", "-c", sshCmd)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		logWriter.Write([]byte("ERROR: SSH-Executor-Fehler: " + err.Error() + "\n"))
		return 1
	}
	return 0
}
