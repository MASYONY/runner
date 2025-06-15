package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJobResult speichert ein beliebiges Ergebnis als result.json im Job-Workdir
func WriteJobResult(jobID string, workDir string, result interface{}) error {
	jobDir := filepath.Join(workDir, jobID)
	os.MkdirAll(jobDir, 0755)
	resultPath := filepath.Join(jobDir, "result.json")
	file, err := os.Create(resultPath)
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen von result.json: %w", err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
