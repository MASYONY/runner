package jobs

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MASYONY/runner/executors"
	"github.com/MASYONY/runner/utils"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v3"
)

type Artifact struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

type CallbackConfig struct {
	URL    string
	Secret string
}

type Job struct {
	JobID     string            `yaml:"job_id"`
	Type      string            `yaml:"type"`
	Executor  string            `yaml:"executor"`
	Product   map[string]string `yaml:"product"`
	Artifacts []Artifact        `yaml:"artifacts"`
	Variables map[string]string `yaml:"variables"`
	Callback  struct {
		URL    string `yaml:"url"`
		Secret string `yaml:"secret"`
	} `yaml:"callback"`
	Status   string `yaml:"-"`
	ExitCode int    `yaml:"-"`
	LogFile  string `yaml:"-"`
}

func generateRandomID() string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func LoadJobFile(path string) (*Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var job Job
	if err := yaml.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	job.JobID = generateRandomID()
	return &job, nil
}

// Neue Funktion zum Laden mehrerer Jobs aus einer YAML-Datei
func LoadJobsFile(path string) ([]*Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var jobs []*Job
	if err := yaml.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}
	for _, job := range jobs {
		job.JobID = generateRandomID()
	}
	return jobs, nil
}

func writeStatusFile(job *Job, jobDir string) {
	statusFile := filepath.Join(jobDir, "status.yaml")
	statusData := struct {
		JobID     string `yaml:"job_id"`
		Status    string `yaml:"status"`
		ExitCode  int    `yaml:"exit_code"`
		LogFile   string `yaml:"log_file"`
		Timestamp string `yaml:"timestamp"`
	}{
		JobID:     job.JobID,
		Status:    job.Status,
		ExitCode:  job.ExitCode,
		LogFile:   job.LogFile,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	data, err := yaml.Marshal(&statusData)
	if err == nil {
		_ = os.WriteFile(statusFile, data, 0644)
	}
}

func RunJob(job *Job, logDir, workDir, defaultCallbackURL, defaultCallbackSecret string) {
	job.Status = "pending"
	job.ExitCode = -1
	jobDir := filepath.Join(workDir, job.JobID)
	os.MkdirAll(jobDir, 0755)
	os.MkdirAll(logDir, 0755)

	// Logfile anlegen
	logFilePath := filepath.Join(logDir, job.JobID+".log")
	logFile, err := os.Create(logFilePath)
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}
	defer logFile.Close()
	job.LogFile = logFilePath

	utils.InfoLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
	utils.ErrorLogger.SetOutput(io.MultiWriter(os.Stderr, logFile))

	// Status-Datei: pending
	writeStatusFile(job, jobDir)

	utils.InfoLogger.Println("Starting job:", job.JobID)
	job.Status = "running"
	writeStatusFile(job, jobDir)

	var exitCode int
	switch job.Executor {
	case "docker":
		exitCode = executors.RunDocker(job.JobID, job.Product, job.Variables, io.MultiWriter(os.Stderr, logFile))
	default:
		utils.ErrorLogger.Printf("Unknown executor %q. Aborted.", job.Executor)
		exitCode = 1
	}

	job.ExitCode = exitCode
	if exitCode == 0 {
		job.Status = "success"
		utils.InfoLogger.Println("Job finished successfully:", job.JobID)
	} else {
		job.Status = "failed"
		utils.ErrorLogger.Println("Job failed:", job.JobID)
	}
	writeStatusFile(job, jobDir)

	// Artifacts kopieren (nur wenn explizit definiert, unterstützt Wildcards)
	jobWorkdir := filepath.Join(workDir, job.JobID)
	if len(job.Artifacts) > 0 {
		for _, artifact := range job.Artifacts {
			artifactPath := artifact.Path
			if !strings.HasPrefix(artifactPath, "mnt/") && !strings.HasPrefix(artifactPath, "mnt\\") {
				artifactPath = filepath.Join("mnt", artifactPath)
			}
			pattern := filepath.Join(jobWorkdir, artifactPath)
			matches, err := filepath.Glob(pattern)
			if err != nil {
				utils.ErrorLogger.Printf("Glob-Fehler für %q: %v", artifact.Path, err)
				continue
			}
			if len(matches) == 0 {
				utils.ErrorLogger.Printf("Kein Artifact gefunden für Pattern: %s", artifact.Path)
			}
			for _, srcPath := range matches {
				destPath := filepath.Join(jobDir, filepath.Base(srcPath))
				err := copyFile(srcPath, destPath)
				if err != nil {
					utils.ErrorLogger.Printf("Error copying artifact %q: %v", srcPath, err)
				} else {
					utils.InfoLogger.Printf("Artifact copied: %s", destPath)
				}
			}
		}
	} else {
		utils.InfoLogger.Println("Keine artifacts im Job definiert – es wird nichts kopiert.")
	}
	// Arbeitsverzeichnis nach dem Kopieren/Job-Ende löschen (nur mnt-Unterordner)
	mntDir := filepath.Join(workDir, job.JobID, "mnt")
	err = os.RemoveAll(mntDir)
	if err != nil {
		utils.ErrorLogger.Printf("Fehler beim Entfernen des Arbeitsverzeichnisses %q: %v", mntDir, err)
	} else {
		utils.InfoLogger.Printf("Arbeitsverzeichnis %q entfernt.", mntDir)
	}

	// Callback-URL aus Job oder global
	callbackURL := job.Callback.URL
	callbackSecret := job.Callback.Secret
	if callbackURL == "" {
		callbackURL = defaultCallbackURL
		callbackSecret = defaultCallbackSecret
	}
	if callbackURL != "" {
		sendCallback(callbackURL, callbackSecret, job)
	}
}

func sendCallback(url, secret string, job *Job) {
	client := resty.New()
	payload := map[string]interface{}{
		"job_id":    job.JobID,
		"status":    job.Status,
		"exit_code": job.ExitCode,
		"log_file":  job.LogFile,
		"artifacts": job.Artifacts,
	}
	req := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload)
	if secret != "" {
		req.SetHeader("Authorization", "Bearer "+secret)
	}
	resp, err := req.Post(url)
	if err != nil {
		utils.ErrorLogger.Printf("Callback error: %v", err)
	} else {
		utils.InfoLogger.Printf("Callback response status: %s", resp.Status())
	}
}

func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	return err
}
