package executors

import (
	"io"
)

// RunSevDesk führt Aktionen gegen die sevDesk-API aus (z.B. Rechnung erstellen, stornieren)
func RunSevDesk(jobID string, product map[string]interface{}, variables map[string]string, logWriter io.Writer) int {
	// TODO: Implementiere sevDesk-API-Calls (z.B. HTTP-Request an REST-API)
	io.WriteString(logWriter, "sevDesk-Executor: Noch nicht implementiert\n")
	return 1
}
