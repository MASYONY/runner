package executors

import (
	"bufio"
	"fmt"
	"io"
	"log"
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

type DockerProduct struct {
	Image        string
	BeforeScript []string
	Script       []string
	Commands     string
	Namespace    string
}

func RunDocker(jobID string, product DockerProduct, variables map[string]string, logWriter io.Writer, useTTY bool) int {
	DefaultInfoLogger := log.New(logWriter, "INFO: ", log.LstdFlags)
	DefaultErrorLogger := log.New(logWriter, "ERROR: ", log.LstdFlags)

	DefaultInfoLogger.Printf("[Docker Executor] Starte Job %s", jobID)

	image := strings.TrimSpace(product.Image)
	if image == "" {
		DefaultErrorLogger.Printf("Docker Executor: Error: No image defined")
		return 1
	}

	// Sammle alle Befehle: before_script, commands, script
	var commands []string
	if len(product.BeforeScript) > 0 {
		commands = append(commands, product.BeforeScript...)
	}
	commandsRaw := strings.TrimSpace(product.Commands)
	if commandsRaw != "" {
		for _, line := range strings.Split(commandsRaw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				commands = append(commands, line)
			}
		}
	}
	if len(product.Script) > 0 {
		commands = append(commands, product.Script...)
	}
	if len(commands) == 0 {
		DefaultErrorLogger.Printf("Docker Executor: Error: No commands to execute")
		return 1
	}

	// Namespace aus product oder variables lesen (optional)
	namespace := "runner"
	if product.Namespace != "" {
		namespace = product.Namespace
	} else if ns, ok := variables["NAMESPACE"]; ok && ns != "" {
		namespace = ns
	}

	containerName := "runner_" + jobID

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

	// Umgebungsvariablen vorbereiten
	env := []string{}
	for key, val := range DefaultJobEnv {
		if key == "JOB_ID" {
			val = jobID
		}
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}
	for key, val := range variables {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}
	env = append(env, fmt.Sprintf("JOB_WORKDIR=%s", containerWorkdir))

	DefaultInfoLogger.Printf("[Docker Executor] Verwende Image: %s", image)
	DefaultInfoLogger.Printf("[Docker Executor] Führe aus: %s", strings.Join(commands, " && "))
	DefaultInfoLogger.Printf("[Docker Executor] Namespace: %s, Containername: %s", namespace, containerName)
	DefaultInfoLogger.Printf("[Docker Executor] Mount: %s -> %s", mntHostDirAbs, containerWorkdir)

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
	if useTTY {
		dockerArgs = append(dockerArgs, "-t")
	}
	for _, e := range env {
		dockerArgs = append(dockerArgs, "--env", e)
	}
	dockerArgs = append(dockerArgs, image, "sh", "-c", strings.Join(commands, " && "))

	cmd := exec.Command("docker", dockerArgs...)
	// Statt direktes logWriter: Output abfangen und mit Logger loggen
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	// Logger-Forwarder
	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			DefaultInfoLogger := log.New(logWriter, "INFO: ", log.LstdFlags)
			DefaultInfoLogger.Println(line)
		}
		close(done)
	}()

	err = cmd.Run()
	pw.Close()
	<-done

	if err != nil {
		DefaultErrorLogger.Printf("[Docker Executor] Fehler: %v", err)
		return 1
	}

	DefaultInfoLogger.Printf("[Docker Executor] Job %s erfolgreich beendet", jobID)
	return 0
}

func getEnv(key string, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
