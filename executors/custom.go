package executors

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/MASYONY/runner/utils"
)

// CustomScriptExecutor mit Interpolation
func RunCustom(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer, workDir string, jobResults map[string]map[string]interface{}, previousJobID string, jobIDMap map[string]string) int {
	var cmdStr string
	if script, ok := product["script"]; ok {
		switch v := script.(type) {
		case []interface{}:
			var lines []string
			for _, s := range v {
				switch val := s.(type) {
				case string:
					lines = append(lines, utils.InterpolateVars(val, workDir, jobResults, previousJobID, jobIDMap, nil))
				case []interface{}:
					for _, inner := range val {
						if str, ok := inner.(string); ok {
							lines = append(lines, utils.InterpolateVars(str, workDir, jobResults, previousJobID, jobIDMap, nil))
						}
					}
				}
			}
			cmdStr = strings.Join(lines, "\n")
		case []string:
			for _, s := range v {
				cmdStr += utils.InterpolateVars(s, workDir, jobResults, previousJobID, jobIDMap, nil) + "\n"
			}
		case string:
			cmdStr = utils.InterpolateVars(v, workDir, jobResults, previousJobID, jobIDMap, nil)
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
	// Interpolation für alle Variablenwerte (auch rekursiv, falls Platzhalter enthalten)
	for k, v := range variables {
		variables[k] = utils.InterpolateVars(v, workDir, jobResults, previousJobID, jobIDMap, nil)
	}
	for k, v := range variables {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// Debug: Logge previousJobID und Interpolation von ${PREVIOUS_JOB_ID}
	logWriter.Write([]byte(fmt.Sprintf("[Custom-Executor-DEBUG] previousJobID: %q\n", previousJobID)))
	logWriter.Write([]byte(fmt.Sprintf("[Custom-Executor-DEBUG] Interpoliert: %q\n", utils.InterpolateVars("${PREVIOUS_JOB_ID}", workDir, jobResults, previousJobID, jobIDMap, nil))))
	if err := cmd.Run(); err != nil {
		logWriter.Write([]byte("ERROR: Custom-Script-Fehler: " + err.Error() + "\n"))
		return 1
	}
	return 0
}
