package executors

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func RunDocker(jobID string, product map[string]string, logWriter io.Writer) int {
	fmt.Fprintf(logWriter, "Docker Executor: Starte Job %s\n", jobID)

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

	fmt.Fprintf(logWriter, "Docker Executor: Start container with image: %s\n", image)
	fmt.Fprintf(logWriter, "Docker Executor: Execute command: %s\n", fullCmd)

	cmd := exec.Command("docker", "run", "--rm", image, "sh", "-c", fullCmd)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(logWriter, "Docker Executor: Error during execution: %v\n", err)
		return 1
	}

	fmt.Fprintf(logWriter, "Docker Executor: Job %s successfully completed\n", jobID)
	return 0
}
