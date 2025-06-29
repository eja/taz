// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"embed"
	"flag"
	"html/template"
	"log"
)

var templates *template.Template
var appLogger *log.Logger

//go:embed assets/*
var embeddedAssets embed.FS

const (
	sessionCookie = "taz_auth"
	appLabel      = "TAZ File Manager"
	appVersion    = "1.6.29"
)

var (
	webHost    = flag.String("web-host", "localhost", "The host address to listen on")
	webPort    = flag.String("web-port", "35248", "The port for the web server")
	password   = flag.String("password", "", "Password for write operations (empty for no auth)")
	rootPath   = flag.String("root", "files", "The root directory for file management")
	logEnabled = flag.Bool("log", false, "Enable logging")
	logFile    = flag.String("log-file", "", "Path to the log file")
	bbsPath    = flag.String("bbs", "", "Path to the BBS database (default: disabled)")
	urlList    stringSlice
)

var externalLinks []ExternalLink
var db *sql.DB
