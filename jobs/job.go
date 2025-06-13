package jobs

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/MASYONY/runner/executors"
	"github.com/MASYONY/runner/utils"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v3"
)

type Artifact struct {
	Path string `yaml:"path`
	Type string `yaml:"type`
}

type CallbackConfig struct {
	URL    string
	Secret string
}

type Job struct {
	JobID     string            `yaml:"job_id`
	Type      string            `yaml:"type`
	Executor  string            `yaml:"executor`
	Product   map[string]string `yaml:"product`
	Artifacts []Artifact        `yaml:"artifacts`
	Callback  struct {
		URL    string `yaml:"url`
		Secret string `yaml:"secret`
	} `yaml:"callback`
	Status   string `yaml:"-`
	ExitCode int    `yaml:"-`
	LogFile  string `yaml:"-`
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

	utils.InfoLogger.Println("Starting job:", job.JobID)
	job.Status = "running"

	var exitCode int
	switch job.Executor {
	case "docker":
		exitCode = executors.RunDocker(job.JobID, job.Product, io.MultiWriter(os.Stderr, logFile))
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

	// Artifacts kopieren
	for _, artifact := range job.Artifacts {
		destPath := filepath.Join(jobDir, filepath.Base(artifact.Path))
		err := copyFile(artifact.Path, destPath)
		if err != nil {
			utils.ErrorLogger.Printf("Error copying artifact %q: %v", artifact.Path, err)
		} else {
			utils.InfoLogger.Printf("Artifact copied: %s", destPath)
		}
	}

	// Status in Datei schreiben
	statusFile := filepath.Join(jobDir, "status.yaml")
	statusData := struct {
		JobID    string `yaml:"job_id`
		Status   string `yaml:"status`
		ExitCode int    `yaml:"exit_code`
		LogFile  string `yaml:"log_file`
	}{
		JobID:    job.JobID,
		Status:   job.Status,
		ExitCode: job.ExitCode,
		LogFile:  job.LogFile,
	}
	data, err := yaml.Marshal(&statusData)
	if err == nil {
		ioutil.WriteFile(statusFile, data, 0644)
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
