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
	JobID     string                 `yaml:"job_id"`
	Type      string                 `yaml:"type"`
	Executor  string                 `yaml:"executor"`
	Product   map[string]interface{} `yaml:"product"`
	Artifacts []Artifact             `yaml:"artifacts"`
	Variables map[string]string      `yaml:"variables"`
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

func RunJob(job *Job, logDir, workDir, defaultCallbackURL, defaultCallbackSecret string, globalBeforeScript []string) {
	// Vereinfachte Proxmox-Job-Syntax (Shortcuts für typische Aktionen)
	if job.Executor == "proxmox" {
		switch job.Type {
		case "lxc_create":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "create"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "kvm_create":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "create"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "lxc_start":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "status/start"
		case "lxc_stop":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "status/stop"
		case "lxc_delete":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "delete"
		case "kvm_start":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "status/start"
		case "kvm_stop":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "status/stop"
		case "kvm_delete":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "delete"
		case "lxc_snapshot":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "snapshot"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "kvm_snapshot":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "snapshot"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		}
	}

	// Weitere Proxmox-Shortcuts (Status, Config, Agent, VNC, Migrate, Clone, Resize, Firewall, Metrics, List)
	if job.Executor == "proxmox" {
		switch job.Type {
		case "kvm_status":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "status/current"
		case "lxc_status":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "status/current"
		case "kvm_config":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "config"
		case "lxc_config":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "config"
		case "kvm_agent_exec":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "agent/exec"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "lxc_agent_exec":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "agent/exec"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "kvm_vncproxy":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "vncproxy"
		case "lxc_vncproxy":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "vncproxy"
		case "kvm_migrate":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "migrate"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "lxc_migrate":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "migrate"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "kvm_clone":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "clone"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "lxc_clone":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "clone"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "kvm_resize":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "resize"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "lxc_resize":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "resize"
			if params, ok := job.Product["params"]; ok {
				job.Product["api_params"] = params
				delete(job.Product, "params")
			}
		case "kvm_firewall":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "firewall"
		case "lxc_firewall":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "firewall"
		case "kvm_metrics":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = "rrddata"
		case "lxc_metrics":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = "rrddata"
		case "kvm_list":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "qemu"
			job.Product["api_command"] = ""
			// List: GET /nodes/{node}/qemu
			job.Product["list_mode"] = true
		case "lxc_list":
			if job.Product == nil {
				job.Product = make(map[string]interface{})
			}
			job.Product["type"] = "lxc"
			job.Product["api_command"] = ""
			job.Product["list_mode"] = true
		}
	}

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
	// Callback beim Start (Status running)
	callbackURL := job.Callback.URL
	callbackSecret := job.Callback.Secret
	if callbackURL == "" {
		callbackURL = defaultCallbackURL
		callbackSecret = defaultCallbackSecret
	}
	if callbackURL != "" {
		sendCallback(callbackURL, callbackSecret, job)
	}

	var exitCode int
	switch job.Executor {
	case "docker":
		// TTY-Option aus Job lesen (Standard: false)
		useTTY := false
		if v, ok := job.Variables["TTY"]; ok && (v == "true" || v == "1") {
			useTTY = true
		}
		// before_script: global + job-spezifisch
		allBefore := append([]string{}, globalBeforeScript...)
		if before, ok := job.Product["before_script"]; ok {
			switch v := before.(type) {
			case []interface{}:
				for _, b := range v {
					if s, ok := b.(string); ok {
						allBefore = append(allBefore, s)
					}
				}
			case string:
				for _, line := range strings.Split(v, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						allBefore = append(allBefore, line)
					}
				}
			}
		}
		var script []string
		if scr, ok := job.Product["script"]; ok {
			switch v := scr.(type) {
			case []interface{}:
				for _, s := range v {
					if str, ok := s.(string); ok {
						script = append(script, str)
					}
				}
			case string:
				for _, line := range strings.Split(v, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						script = append(script, line)
					}
				}
			}
		}
		commands := ""
		if cmd, ok := job.Product["commands"]; ok {
			switch v := cmd.(type) {
			case []interface{}:
				var lines []string
				for _, s := range v {
					if str, ok := s.(string); ok {
						lines = append(lines, str)
					}
				}
				commands = strings.Join(lines, "\n")
			case string:
				commands = v
			}
		}
		// Felder für DockerProduct korrekt initialisieren
		var image, namespace string
		if img, ok := job.Product["image"]; ok {
			image, _ = img.(string)
		}
		namespace = "runner"
		if ns, ok := job.Product["namespace"]; ok {
			namespace, _ = ns.(string)
		}
		// Baue ein struct für den Executor
		type DockerProduct struct {
			Image        string
			BeforeScript []string
			Script       []string
			Commands     string
			Namespace    string
		}
		product := executors.DockerProduct{
			Image:        image,
			BeforeScript: allBefore,
			Script:       script,
			Commands:     commands,
			Namespace:    namespace,
		}
		exitCode = executors.RunDocker(job.JobID, product, job.Variables, io.MultiWriter(os.Stderr, logFile), useTTY)
	case "custom":
		exitCode = executors.RunCustom(job.JobID, job.Product, job.Variables, io.MultiWriter(os.Stderr, logFile))
	case "local":
		exitCode = executors.RunLocal(job.JobID, job.Product, job.Variables, io.MultiWriter(os.Stderr, logFile))
	case "ssh":
		exitCode = executors.RunSSH(job.JobID, job.Product, job.Variables, io.MultiWriter(os.Stderr, logFile))
	case "proxmox":
		exitCode = executors.RunProxmox(job.JobID, job.Product, job.Variables, io.MultiWriter(os.Stderr, logFile))
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
	callbackURL = job.Callback.URL
	callbackSecret = job.Callback.Secret
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
