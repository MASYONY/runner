import zipfile

# Neues Runner Projekt mit Punkten 1-7 umgesetzt

enhanced_project_files = {
    "runner/main.go": '''package main

import "github.com/dein-user/runner/cmd"

func main() {
    cmd.Execute()
}
''',
    "runner/go.mod": '''module github.com/dein-user/runner

go 1.21

require (
    github.com/spf13/cobra v1.7.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/go-resty/resty/v2 v2.7.0
)
''',
    "runner/cmd/root.go": '''
package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/dein-user/runner/jobs"
    "github.com/dein-user/runner/utils"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

var (
    file      string
    config    string
    logDir    string
    workDir   string
    callback  string
    debugMode bool
)

type RunnerConfig struct {
    DefaultLogDir string ` + "`yaml:\"default_log_dir\"`" + `
    DefaultWorkDir string ` + "`yaml:\"default_work_dir\"`" + `
    Callback      struct {
        URL    string ` + "`yaml:\"url\"`" + `
        Secret string ` + "`yaml:\"secret\"`" + `
    } ` + "`yaml:\"callback\"`" + `
}

var runnerConfig RunnerConfig

var rootCmd = &cobra.Command{
    Use:   "runner",
    Short: "Modularer Runner für Jobs aus YAML mit erweitertem Feature-Set",
}

var runCmd = &cobra.Command{
    Use:   "run --file job.yaml",
    Short: "Führe einen Job aus",
    Run: func(cmd *cobra.Command, args []string) {
        err := loadConfig(config)
        if err != nil {
            fmt.Println("Fehler beim Laden der Config:", err)
            os.Exit(1)
        }

        if logDir == "" {
            logDir = runnerConfig.DefaultLogDir
            if logDir == "" {
                logDir = "./logs"
            }
        }
        if workDir == "" {
            workDir = runnerConfig.DefaultWorkDir
            if workDir == "" {
                workDir = "./workdir"
            }
        }

        jobDef, err := jobs.LoadJobFile(file)
        if err != nil {
            fmt.Println("Failed to load job:", err)
            os.Exit(1)
        }

        // Job ausführen mit erweiterten Optionen
        jobs.RunJob(jobDef, logDir, workDir, runnerConfig.Callback.URL, runnerConfig.Callback.Secret)
    },
}

func loadConfig(path string) error {
    if path == "" {
        return nil
    }
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return yaml.Unmarshal(data, &runnerConfig)
}

func init() {
    rootCmd.AddCommand(runCmd)
    runCmd.Flags().StringVarP(&file, "file", "f", "", "Pfad zur Job-YAML")
    runCmd.MarkFlagRequired("file")

    runCmd.Flags().StringVarP(&config, "config", "c", "", "Pfad zur Runner-Konfigurationsdatei (YAML)")
    runCmd.Flags().StringVar(&logDir, "log-dir", "", "Verzeichnis für Job-Logs (überschreibt config)")
    runCmd.Flags().StringVar(&workDir, "workdir", "", "Arbeitsverzeichnis für Job-Artifacts (überschreibt config)")
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
''',
    "runner/jobs/job.go": '''package jobs

import (
    "fmt"
    "io"
    "io/ioutil"
    "math/rand"
    "os"
    "os/exec"
    "path/filepath"
    "time"

    "github.com/dein-user/runner/executors"
    "github.com/dein-user/runner/utils"
    "gopkg.in/yaml.v3"
    "github.com/go-resty/resty/v2"
)

type Artifact struct {
    Path string ` + "`yaml:\"path\"`" + `
    Type string ` + "`yaml:\"type\"`" + `
}

type CallbackConfig struct {
    URL    string
    Secret string
}

type Job struct {
    JobID     string            ` + "`yaml:\"job_id\"`" + `
    Type      string            ` + "`yaml:\"type\"`" + `
    Executor  string            ` + "`yaml:\"executor\"`" + `
    Product   map[string]string ` + "`yaml:\"product\"`" + `
    Artifacts []Artifact        ` + "`yaml:\"artifacts\"`" + `
    Callback  struct {
        URL    string ` + "`yaml:\"url\"`" + `
        Secret string ` + "`yaml:\"secret\"`" + `
    } ` + "`yaml:\"callback\"`" + `
    Status   string            ` + "`yaml:\"-\"`" + `
    ExitCode int               ` + "`yaml:\"-\"`" + `
    LogFile  string            ` + "`yaml:\"-\"`" + `
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
        exitCode = executors.RunDocker(job.JobID)
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
        JobID    string ` + "`yaml:\"job_id\"`" + `
        Status   string ` + "`yaml:\"status\"`" + `
        ExitCode int    ` + "`yaml:\"exit_code\"`" + `
        LogFile  string ` + "`yaml:\"log_file\"`" + `
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
        "job_id":   job.JobID,
        "status":   job.Status,
        "exit_code": job.ExitCode,
        "log_file": job.LogFile,
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
''',
    "runner/executors/docker.go": '''package executors

import (
    "fmt"
    "os/exec"
)

func RunDocker(jobID string) int {
    fmt.Printf("[Docker Executor] Running job %s...\\n", jobID)
    cmd := exec.Command("echo", fmt.Sprintf("Provisioning job %s", jobID))
    out, err := cmd.CombinedOutput()
    fmt.Println(string(out))
    if err != nil {
        fmt.Printf("[Docker Executor] Error: %v\\n", err)
        return 1
    }
    fmt.Println("[Docker Executor] Done.")
    return 0
}
''',
    "runner/utils/logger.go": '''package utils

import (
    "io"
    "log"
    "os"
)

var (
    InfoLogger  = log.New(os.Stdout, "INFO: ", log.LstdFlags)
    ErrorLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
)

func SetOutput(w io.Writer) {
    InfoLogger.SetOutput(w)
    ErrorLogger.SetOutput(w)
}
''',
    "runner/job-example.yaml": '''type: provisioning
executor: docker
product:
  name: "TestSite"
  domain: example.com
  user: fabian
artifacts:
  - path: ./README.md
    type: log
callback:
  url: ""
  secret: ""
''',
    "runner/jobs/multi-jobs.yaml": '''- type: provisioning
  executor: docker
  product:
    name: MultiTest1
    domain: multi1.example.com
    user: fabian
  artifacts:
    - path: ./README.md
      type: log
- type: provisioning
  executor: docker
  product:
    name: MultiTest2
    domain: multi2.example.com
    user: fabian
  artifacts:
    - path: ./README.md
      type: log
'''
}

# Schreibe neues ZIP mit aktualisierten Dateien
enhanced_zip_path = "runner-enhanced.zip"
with zipfile.ZipFile(enhanced_zip_path, "w") as zipf:
    for filepath, content in enhanced_project_files.items():
        zipf.writestr(filepath, content)

enhanced_zip_path
