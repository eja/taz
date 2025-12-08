// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	initOptions()

	var logOutput io.Writer = io.Discard
	if options.LogEnabled {
		if options.LogFile != "" {
			f, err := os.OpenFile(options.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("Failed to open log file: %v", err)
			}
			logOutput = f
		} else {
			logOutput = os.Stderr
		}
	}
	appLogger = log.New(logOutput, "", log.LstdFlags)

	addr := fmt.Sprintf("%s:%d", options.WebHost, options.WebPort)

	if err := os.MkdirAll(options.RootPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create root directory '%s': %v", options.RootPath, err)
	}

	startNetworkServices()

	setupTemplates()
	setupRoutes()

	appLogger.Printf("Starting TAZ file manager on http://%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
