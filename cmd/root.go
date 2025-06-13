package cmd

import (
	"fmt"
	"os"

	"github.com/MASYONY/runner/jobs"
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
	DefaultLogDir  string `yaml:"default_log_dir"`
	DefaultWorkDir string `yaml:"default_work_dir"`
	Callback       struct {
		URL    string `yaml:"url"`
		Secret string `yaml:"secret"`
	} `yaml:"callback"`
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
