// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"html/template"
	"io/fs"
	"log"
	"strings"
)

var templateFuncs = template.FuncMap{
	"split": func(s string) []string { return strings.Split(s, "/") },
	"join":  func(s []string) string { return strings.Join(s, "/") },
	"slice": func(s []string, i, j int) []string { return s[i:j] },
	"add":   func(i, j int) int { return i + j },
}

func setupTemplates() {
	assetsFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		log.Fatalf("Failed to create sub FS for assets: %v", err)
	}
	templates = template.Must(template.New("").Funcs(templateFuncs).ParseFS(assetsFS, "*.html"))
}
