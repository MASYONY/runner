package cmd

import (
	"fmt"
	"os"

	"github.com/MASYONY/runner/jobs"
	"github.com/MASYONY/runner/utils"
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
	DefaultLogDir      string   `yaml:"default_log_dir"`
	DefaultWorkDir     string   `yaml:"default_work_dir"`
	GlobalBeforeScript []string `yaml:"before_script"`
	Callback           struct {
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
	Use:   "run <job.yaml>",
	Short: "Führe einen Job aus",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file = args[0]
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
		jobs.RunJob(jobDef, logDir, workDir, runnerConfig.Callback.URL, runnerConfig.Callback.Secret, runnerConfig.GlobalBeforeScript)
	},
}

var runMultiCmd = &cobra.Command{
	Use:   "run-multi <multi-jobs.yaml>",
	Short: "Führe mehrere Jobs aus einer YAML-Liste aus",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file = args[0]
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

		jobsList, err := jobs.LoadJobsFile(file)
		if err != nil {
			fmt.Println("Failed to load jobs:", err)
			os.Exit(1)
		}

		for i, jobDef := range jobsList {
			fmt.Printf("\n--- Starte Job %d: %s ---\n", i+1, jobDef.Type)
			jobs.RunJob(jobDef, logDir, workDir, runnerConfig.Callback.URL, runnerConfig.Callback.Secret, runnerConfig.GlobalBeforeScript)
		}
	},
}

func loadConfig(path string) error {
	if path == "" {
		// Fallback: config.yaml im aktuellen Verzeichnis
		if _, err := os.Stat("config.yaml"); err == nil {
			path = "config.yaml"
		}
	}
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
	runCmd.Flags().StringVarP(&config, "config", "c", "", "Pfad zur Runner-Konfigurationsdatei (YAML)")
	runCmd.Flags().StringVar(&logDir, "log-dir", "", "Verzeichnis für Job-Logs (überschreibt config)")
	runCmd.Flags().StringVar(&workDir, "workdir", "", "Arbeitsverzeichnis für Job-Artifacts (überschreibt config)")

	// Neuen Multi-Job-Command registrieren
	rootCmd.AddCommand(runMultiCmd)
	runMultiCmd.Flags().StringVarP(&config, "config", "c", "", "Pfad zur Runner-Konfigurationsdatei (YAML)")
	runMultiCmd.Flags().StringVar(&logDir, "log-dir", "", "Verzeichnis für Job-Logs (überschreibt config)")
	runMultiCmd.Flags().StringVar(&workDir, "workdir", "", "Arbeitsverzeichnis für Job-Artifacts (überschreibt config)")
}

func Execute() {
	utils.InitSocketLogging()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
