package utils

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
