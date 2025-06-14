package executors

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Standardisierte Umgebungsvariablen, die jeder Job mitbekommt
var DefaultJobEnv = map[string]string{
	"JOB_ID":          "", // wird zur Laufzeit gesetzt
	"RUNNER_ID":       getEnv("RUNNER_ID", "unknown"),
	"RUNNER_HOSTNAME": getEnv("RUNNER_HOSTNAME", "localhost"),
	"RUNNER_WORKDIR":  getEnv("RUNNER_WORK_DIR", "/runner/workdir"),
	"RUNNER_LOG_DIR":  getEnv("RUNNER_LOG_DIR", "/runner/logs"),
}

func RunDocker(jobID string, product map[string]string, logWriter io.Writer) int {
	fmt.Fprintf(logWriter, "[Docker Executor] Starte Job %s\n", jobID)

	image := strings.TrimSpace(product["image"])
	if image == "" {
		fmt.Fprintf(logWriter, "Docker Executor: Error: No image defined\n")
		return 1
	}

	commandsRaw := strings.TrimSpace(product["commands"])
	if commandsRaw == "" {
		fmt.Fprintf(logWriter, "Docker Executor: Error: No commands defined\n")
		return 1
	}

	// Commands in Zeilen aufteilen
	lines := strings.Split(commandsRaw, "\n")
	var commands []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			commands = append(commands, line)
		}
	}

	// Alle commands als eine Shell-Zeile mit &&
	fullCmd := strings.Join(commands, " && ")

	// Umgebungsvariablen vorbereiten
	env := []string{}
	for key, val := range DefaultJobEnv {
		if key == "JOB_ID" {
			val = jobID
		}
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}

	fmt.Fprintf(logWriter, "[Docker Executor] Verwende Image: %s\n", image)
	fmt.Fprintf(logWriter, "[Docker Executor] Führe aus: %s\n", fullCmd)

	// Docker Socket für Docker-in-Docker ermöglichen
	dockerSock := "/var/run/docker.sock"
	dockerArgs := []string{
		"run", "--rm",
		"-v", dockerSock + ":" + dockerSock,
		"--env", fmt.Sprintf("JOB_ID=%s", jobID),
	}
	for _, e := range env {
		dockerArgs = append(dockerArgs, "--env", e)
	}
	dockerArgs = append(dockerArgs, image, "sh", "-c", fullCmd)

	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(logWriter, "[Docker Executor] Fehler: %v\n", err)
		return 1
	}

	fmt.Fprintf(logWriter, "[Docker Executor] Job %s erfolgreich beendet\n", jobID)
	return 0
}

func getEnv(key string, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
