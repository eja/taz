// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
	"strings"
)

func main() {
	flag.Var(&urlList, "url", "Link to display on root page. Format: 'Name|URL'. Can be used multiple times.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Copyright: 2025 by Ubaldo Porcheddu <ubaldo@eja.it>\nVersion: %s\nUsage: %s [options]\n\n", appVersion, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var logOutput io.Writer = io.Discard
	if *logEnabled {
		if *logFile != "" {
			f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("Failed to open log file: %v", err)
			}
			logOutput = f
		} else {
			logOutput = os.Stderr
		}
	}
	appLogger = log.New(logOutput, "", log.LstdFlags)

	addr := fmt.Sprintf("%s:%s", *webHost, *webPort)

	for _, entry := range urlList {
		parts := strings.SplitN(entry, "|", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
			externalLinks = append(externalLinks, ExternalLink{Name: strings.TrimSpace(parts[0]), URL: strings.TrimSpace(parts[1])})
		} else if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" {
			externalLinks = append(externalLinks, ExternalLink{Name: strings.TrimSpace(parts[0]), URL: strings.TrimSpace(parts[0])})
		}
	}

	if err := os.MkdirAll(*rootPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create root directory '%s': %v", *rootPath, err)
	}

	if *bbsPath != "" {
		var err error
		db, err = sql.Open("sqlite", *bbsPath)
		if err != nil {
			log.Fatalf("Failed to connect to BBS database: %v", err)
		}

		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS bbs_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
		if err != nil {
			log.Fatalf("Failed to create BBS table: %v", err)
		}
	}

	setupTemplates()
	setupRoutes()

	appLogger.Printf("Starting TAZ file manager on http://%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
