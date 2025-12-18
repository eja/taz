// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type logFilter struct {
	w io.Writer
}

func (f *logFilter) Write(p []byte) (int, error) {
	s := string(p)
	if strings.Contains(s, "http: TLS handshake error") && strings.Contains(s, "remote error") {
		return len(p), nil
	}
	return f.w.Write(p)
}

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

	if err := os.MkdirAll(options.RootPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create root directory '%s': %v", options.RootPath, err)
	}
	if err := os.MkdirAll(options.SystemPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create system directory '%s': %v", options.SystemPath, err)
	}

	addr := fmt.Sprintf("%s:%d", options.WebHost, options.WebPort)

	startNetworkServices()

	setupTemplates()
	setupRoutes()

	cert, err := getCertificate(filepath.Join(options.SystemPath, "certificate.pem"))
	if err != nil {
		log.Fatalf("Failed to prepare certificate: %v", err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	mux := &muxListener{
		Listener: ln,
		config: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	server := &http.Server{
		Handler:  nil,
		ErrorLog: log.New(&logFilter{w: logOutput}, "", log.LstdFlags),
	}

	startDiscovery()

	appLogger.Printf("Starting TAZ file manager on http://%s", addr)
	if err := server.Serve(mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
