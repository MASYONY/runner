package executors

import (
	"io"

	"github.com/MASYONY/runner/utils"
)

// RunLexware mit Interpolation
func RunLexware(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer, workDir string, jobResults map[string]map[string]interface{}, previousJobID string, jobIDMap map[string]string) int {
	// Beispiel: Interpolation für alle String-Felder in product und variables
	for k, v := range product {
		if str, ok := v.(string); ok {
			product[k] = utils.InterpolateVars(str, workDir, jobResults, previousJobID, jobIDMap, nil)
		}
	}
	// Interpolation für alle Variablenwerte (rekursiv, falls Platzhalter enthalten)
	for k, v := range variables {
		variables[k] = utils.InterpolateVars(v, workDir, jobResults, previousJobID, jobIDMap, nil)
	}
	io.WriteString(logWriter, "Lexware-Executor: Noch nicht implementiert\n")
	return 1
}
