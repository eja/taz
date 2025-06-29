// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"strings"
)

type stringSlice []string

func (i *stringSlice) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type ExternalLink struct {
	Name string
	URL  string
}

type FileInfo struct {
	Name    string
	Path    string
	Isdir   bool
	Size    string
	ModTime string
}

type PageData struct {
	Title             string
	CurrentPath       string
	ParentPath        string
	Files             []FileInfo
	Message           string
	Error             string
	PasswordProtected bool
	IsAuthenticated   bool
	ExternalLinks     []ExternalLink
}

type EditPageData struct {
	Title      string
	Path       string
	ParentPath string
	Content    string
}
