// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"embed"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"
)

const (
	sessionCookie = "taz_auth"
	appLabel      = "TAZ"
	appVersion    = "1.12.14"
)

//go:embed assets
var embeddedAssets embed.FS

var (
	externalLinks []ExternalLink
	templates     *template.Template
	appLogger     *log.Logger
	options       Options
	urlList       stringSlice
	dhcpList      stringSlice
	uptime        = time.Now()
)

type Options struct {
	WebHost        string   `json:"web_host"`
	WebPort        int      `json:"web_port"`
	Password       string   `json:"password"`
	RootPath       string   `json:"root_path"`
	SystemPath     string   `json:"system_path"`
	LogEnabled     bool     `json:"log_enabled"`
	LogFile        string   `json:"log_file"`
	BBSPath        string   `json:"bbs_path"`
	URLs           []string `json:"urls"`
	DHCPInterfaces []string `json:"dhcp_interfaces"`
	DNS            string   `json:"dns"`
}

func initOptions() {
	options = Options{
		WebHost:  "localhost",
		WebPort:  35248,
		RootPath: "files",
	}

	configFile := flag.String("config", "", "Path to JSON config file")
	webHost := flag.String("web-host", options.WebHost, "The host address to listen on")
	webPort := flag.Int("web-port", options.WebPort, "The port for the web server")
	password := flag.String("password", options.Password, "Password for write operations (empty for no auth)")
	rootPath := flag.String("root", options.RootPath, "The root directory for file management")
	sysPath := flag.String("sys", "", "Path to the system directory (defaults to <root>/sys)")
	logEnabled := flag.Bool("log", options.LogEnabled, "Enable logging")
	logFile := flag.String("log-file", options.LogFile, "Path to the log file")
	flag.Var(&urlList, "url", "Link to display on root page. Format: 'Name|URL'. Repeatable.")
	flag.Var(&dhcpList, "dhcp", "DHCP interface and subnet (e.g., 'wlan0:10.35.2.0/24'). Repeatable.")
	dns := flag.String("dns", "", "Enable DNS sinkhole. Optionally provide an upstream IP (e.g., '8.8.8.8').")

	flag.Parse()

	isFlagSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		isFlagSet[f.Name] = true
	})

	if *configFile != "" {
		data, err := ioutil.ReadFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to read config file: %v", err)
		}
		if err := json.Unmarshal(data, &options); err != nil {
			log.Fatalf("Failed to parse config file: %v", err)
		}
	}

	if isFlagSet["web-host"] {
		options.WebHost = *webHost
	}
	if isFlagSet["web-port"] {
		options.WebPort = *webPort
	}
	if isFlagSet["password"] {
		options.Password = *password
	}
	if isFlagSet["root"] {
		options.RootPath = *rootPath
	}
	if isFlagSet["sys"] {
		options.SystemPath = *sysPath
	}
	if isFlagSet["log"] {
		options.LogEnabled = *logEnabled
	}
	if isFlagSet["log-file"] {
		options.LogFile = *logFile
	}
	if isFlagSet["url"] {
		options.URLs = urlList
	}
	if isFlagSet["dhcp"] {
		options.DHCPInterfaces = dhcpList
	}
	if isFlagSet["dns"] {
		options.DNS = *dns
	}

	for _, entry := range options.URLs {
		parts := strings.SplitN(entry, "|", 2)
		var name, url string
		if len(parts) == 2 {
			name = strings.TrimSpace(parts[0])
			url = strings.TrimSpace(parts[1])
		} else if len(parts) == 1 {
			name = strings.TrimSpace(parts[0])
			url = name
		}

		if name != "" && url != "" {
			externalLinks = append(externalLinks, ExternalLink{Name: name, URL: url})
		}
	}

	if options.SystemPath == "" {
		options.SystemPath = filepath.Join(options.RootPath, "sys")
	}

	options.BBSPath = filepath.Join(options.SystemPath, "bbs.db")
}
