package executors

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/MASYONY/runner/utils"
)

// SSHExecutor mit Interpolation
func RunSSH(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer, workDir string, jobResults map[string]map[string]interface{}, previousJobID string, jobIDMap map[string]string) int {
	host, ok := product["host"].(string)
	if !ok || host == "" {
		logWriter.Write([]byte("ERROR: Kein SSH-Host im Job definiert\n"))
		return 1
	}
	host = utils.InterpolateVars(host, workDir, jobResults, previousJobID, jobIDMap, nil)
	user := "root"
	if u, ok := product["user"].(string); ok && u != "" {
		user = utils.InterpolateVars(u, workDir, jobResults, previousJobID, jobIDMap, nil)
	}
	var cmdStr string
	if commands, ok := product["commands"]; ok {
		switch v := commands.(type) {
		case []interface{}:
			var lines []string
			for _, s := range v {
				if str, ok := s.(string); ok {
					lines = append(lines, utils.InterpolateVars(str, workDir, jobResults, previousJobID, jobIDMap, nil))
				}
			}
			cmdStr = strings.Join(lines, "\n")
		case string:
			cmdStr = utils.InterpolateVars(v, workDir, jobResults, previousJobID, jobIDMap, nil)
		}
	}
	if strings.TrimSpace(cmdStr) == "" {
		logWriter.Write([]byte("ERROR: Keine commands im Job definiert\n"))
		return 1
	}
	// Interpolation für alle Variablenwerte (rekursiv, falls Platzhalter enthalten)
	for k, v := range variables {
		variables[k] = utils.InterpolateVars(v, workDir, jobResults, previousJobID, jobIDMap, nil)
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
