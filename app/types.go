// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

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
	IsMap   bool
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
	HasBBS            bool
}

type EditPageData struct {
	Title      string
	Path       string
	ParentPath string
	Content    string
}

type BBSMessage struct {
	ID        int    `json:"-"`
	Message   string `json:"message"`
	CreatedAt string `json:"time"`
}

type BBSPageData struct {
	Title       string
	Messages    []BBSMessage
	CurrentPage int
	TotalPages  int
	HasPrevious bool
	HasNext     bool
	Pages       []int
}
