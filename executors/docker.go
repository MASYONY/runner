package executors

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func RunDocker(jobID string, product map[string]string, variables map[string]string, logWriter io.Writer) int {
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

	// Namespace aus product oder variables lesen (optional)
	namespace := "runner" // Default Namespace ist jetzt 'runner'
	if ns, ok := product["namespace"]; ok && ns != "" {
		namespace = ns
	} else if ns, ok := variables["NAMESPACE"]; ok && ns != "" {
		namespace = ns
	}

	containerName := "runner_" + jobID

	// Arbeitsverzeichnis für den Job (Host und Container)
	hostWorkdir := os.Getenv("RUNNER_WORKDIR")
	if hostWorkdir == "" {
		hostWorkdir = "./workdir"
	}
	jobHostDir := hostWorkdir + string(os.PathSeparator) + jobID
	mntHostDir := filepath.Join(jobHostDir, "mnt")
	mntHostDirAbs, err := filepath.Abs(mntHostDir)
	if err != nil {
		fmt.Fprintf(logWriter, "[Docker Executor] Fehler beim Ermitteln des absoluten Pfads: %v\n", err)
		return 1
	}
	containerWorkdir := "/runner/jobworkdir"
	_ = os.MkdirAll(mntHostDirAbs, 0755)

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
	// Zusätzliche Variablen aus YAML hinzufügen
	for key, val := range variables {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}
	// Arbeitsverzeichnis im Container als Umgebungsvariable
	env = append(env, fmt.Sprintf("JOB_WORKDIR=%s", containerWorkdir))

	fmt.Fprintf(logWriter, "[Docker Executor] Verwende Image: %s\n", image)
	fmt.Fprintf(logWriter, "[Docker Executor] Führe aus: %s\n", fullCmd)
	fmt.Fprintf(logWriter, "[Docker Executor] Namespace: %s, Containername: %s\n", namespace, containerName)
	fmt.Fprintf(logWriter, "[Docker Executor] Mount: %s -> %s\n", mntHostDirAbs, containerWorkdir)

	// Docker Socket für Docker-in-Docker ermöglichen
	dockerSock := "/var/run/docker.sock"
	dockerArgs := []string{
		"run", "--rm",
		"--name", containerName,
		"-v", dockerSock + ":" + dockerSock,
		"-v", mntHostDirAbs + ":" + containerWorkdir,
		"--label", fmt.Sprintf("namespace=%s", namespace),
		"--env", fmt.Sprintf("JOB_ID=%s", jobID),
	}
	for _, e := range env {
		dockerArgs = append(dockerArgs, "--env", e)
	}
	dockerArgs = append(dockerArgs, image, "sh", "-c", fullCmd)

	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	err = cmd.Run()
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
