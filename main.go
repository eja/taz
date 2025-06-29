// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed assets/*
var embeddedAssets embed.FS

var templates *template.Template
var appLogger *log.Logger

type stringSlice []string

func (i *stringSlice) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

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
	urlList    stringSlice
)

var externalLinks []ExternalLink

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

var templateFuncs = template.FuncMap{
	"split": func(s string) []string { return strings.Split(s, "/") },
	"join":  func(s []string) string { return strings.Join(s, "/") },
	"slice": func(s []string, i, j int) []string { return s[i:j] },
	"add":   func(i, j int) int { return i + j },
}

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

	assetsFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		log.Fatalf("Failed to create sub FS for assets: %v", err)
	}
	templates = template.Must(template.New("").Funcs(templateFuncs).ParseFS(assetsFS, "*.html"))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assetsFS))))

	http.HandleFunc("/", fileManagerHandler)
	http.HandleFunc("/download", downloadHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/edit", editHandler)

	appLogger.Printf("Starting TAZ file manager on http://%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getPasswordHash() string {
	if *password == "" {
		return ""
	}
	hasher := sha256.New()
	hasher.Write([]byte(*password))
	return hex.EncodeToString(hasher.Sum(nil))
}

func requireAuth(next http.HandlerFunc, requireWrite bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if *password == "" {
			next(w, r)
			return
		}

		if requireWrite {
			cookie, err := r.Cookie(sessionCookie)
			if err != nil || !isCookieValid(cookie.Value) {
				returnPath := r.URL.Query().Get("path")
				if fileParam := r.URL.Query().Get("file"); fileParam != "" {
					dir := filepath.Dir(fileParam)
					if dir == "." {
						returnPath = ""
					} else {
						returnPath = filepath.ToSlash(dir)
					}
				}
				http.Redirect(w, r, "/?path="+url.QueryEscape(returnPath), http.StatusSeeOther)
				return
			}
		}

		next(w, r)
	}
}

func isCookieValid(token string) bool {
	if *password == "" {
		return false
	}
	expectedToken := getPasswordHash()
	return subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	pass := r.FormValue("password")
	if subtle.ConstantTimeCompare([]byte(pass), []byte(*password)) == 1 {
		appLogger.Printf("Successful login from %s", r.RemoteAddr)
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    getPasswordHash(),
			Path:     "/",
			HttpOnly: true,
		})
		http.Redirect(w, r, "/?path="+url.QueryEscape(r.URL.Query().Get("path")), http.StatusSeeOther)
		return
	}

	appLogger.Printf("Failed login attempt from %s", r.RemoteAddr)
	http.Redirect(w, r, "/?path="+url.QueryEscape(r.URL.Query().Get("path"))+"&err=Invalid+password", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	appLogger.Printf("Logout from %s", r.RemoteAddr)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/?path="+url.QueryEscape(r.URL.Query().Get("path")), http.StatusSeeOther)
}

func fileManagerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		requireAuth(handlePostRequest, true)(w, r)
		return
	}
	requireAuth(handleGetRequest, false)(w, r)
}

func handleGetRequest(w http.ResponseWriter, r *http.Request) {
	var relativePath string
	relativePath = r.URL.Query().Get("path")
	if relativePath == "" {
		relativePath = "."
	}

	absPath, err := getSafePath(relativePath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		log.Printf("Error: Invalid path access attempt: %s", relativePath)
		return
	}

	renderPage(w, r, absPath, relativePath)
}

func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil && err != http.ErrNotMultipart {
		http.Error(w, "Error parsing form", http.StatusInternalServerError)
		return
	}
	if r.MultipartForm == nil {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error parsing form", http.StatusInternalServerError)
			return
		}
	}
	relativePath := r.FormValue("path")

	if relativePath == "" {
		relativePath = "."
	}

	absPath, err := getSafePath(relativePath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		log.Printf("Error: Invalid path access attempt: %s", relativePath)
		return
	}

	action := r.FormValue("action")
	if action == "createtxt" {
		handleCreateTxt(w, r, absPath)
		return
	}

	var message, errorMsg string
	switch action {
	case "upload":
		message, errorMsg = handleUpload(r, absPath)
	case "mkdir":
		message, errorMsg = handleMkdir(r, absPath)
	case "delete":
		message, errorMsg = handleDelete(r)
	case "rename":
		message, errorMsg = handleRename(r)
	}
	redirectURL := fmt.Sprintf("/?path=%s&msg=%s&err=%s",
		template.URLQueryEscaper(relativePath),
		template.URLQueryEscaper(message),
		template.URLQueryEscaper(errorMsg),
	)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handleSaveFile(w, r)
		} else {
			handleShowEditor(w, r)
		}
	}, true)(w, r)
}

func handleShowEditor(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Query().Get("file")
	safePath, err := getSafePath(relativePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		http.Error(w, "Could not read file", http.StatusInternalServerError)
		return
	}

	data := EditPageData{
		Title:      "Edit " + filepath.Base(relativePath),
		Path:       relativePath,
		ParentPath: filepath.ToSlash(filepath.Dir(relativePath)),
		Content:    string(content),
	}
	templates.ExecuteTemplate(w, "edit.html", data)
}

func handleSaveFile(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	relativePath := r.FormValue("path")
	content := r.FormValue("content")
	safePath, err := getSafePath(relativePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	os.WriteFile(safePath, []byte(content), 0644)
	redirectURL := fmt.Sprintf("/?path=%s&msg=%s",
		url.QueryEscape(filepath.ToSlash(filepath.Dir(relativePath))),
		url.QueryEscape("Saved "+filepath.Base(relativePath)),
	)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Query().Get("file")
	if relativePath == "" {
		http.Error(w, "No file specified", http.StatusBadRequest)
		return
	}

	absPath, err := getSafePath(relativePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found or is a directory", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, absPath)
}

func renderPage(w http.ResponseWriter, r *http.Request, absPath, relativePath string) {
	dirEntries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, "Could not read directory", http.StatusInternalServerError)
		return
	}
	var files []FileInfo
	for _, entry := range dirEntries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    filepath.ToSlash(filepath.Join(relativePath, entry.Name())),
			Isdir:   entry.IsDir(),
			Size:    formatFileSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Isdir != files[j].Isdir {
			return files[i].Isdir
		}
		return files[i].Name < files[j].Name
	})

	parentPath := ""
	if relativePath != "." && relativePath != "" {
		parentPath = filepath.ToSlash(filepath.Dir(relativePath))
	}

	var isAuthenticated bool
	cookie, err := r.Cookie(sessionCookie)
	if err == nil {
		isAuthenticated = isCookieValid(cookie.Value)
	}

	data := PageData{
		Title:             appLabel,
		CurrentPath:       relativePath,
		ParentPath:        parentPath,
		Files:             files,
		Message:           r.URL.Query().Get("msg"),
		Error:             r.URL.Query().Get("err"),
		PasswordProtected: *password != "",
		IsAuthenticated:   isAuthenticated,
	}

	if relativePath == "." || relativePath == "" {
		data.ExternalLinks = externalLinks
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "index.html", data)
}

func handleUpload(r *http.Request, destPath string) (string, string) {
	form := r.MultipartForm
	if form == nil {
		return "", "No files uploaded."
	}

	files := form.File["files"]
	if len(files) == 0 {
		return "", "No files selected."
	}

	var uploadedFiles []string
	for _, handler := range files {
		srcFile, err := handler.Open()
		if err != nil {
			return "", fmt.Sprintf("Failed to open file '%s'.", handler.Filename)
		}
		defer srcFile.Close()

		cleanFilename := filepath.Base(handler.Filename)
		if strings.ContainsAny(cleanFilename, `/\:*?"<>|`) {
			continue
		}

		destFile, err := os.Create(filepath.Join(destPath, cleanFilename))
		if err != nil {
			return "", fmt.Sprintf("Could not create file '%s' on server.", cleanFilename)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return "", fmt.Sprintf("Failed to save file '%s'.", cleanFilename)
		}
		uploadedFiles = append(uploadedFiles, cleanFilename)
	}

	if len(uploadedFiles) == 0 {
		return "", "No valid files were uploaded."
	}

	return fmt.Sprintf("Uploaded: %s", strings.Join(uploadedFiles, ", ")), ""
}

func handleMkdir(r *http.Request, currentPath string) (string, string) {
	dirName := r.FormValue("dirname")
	appLogger.Printf("MKDIR by %s: creating directory '%s' in '%s'", r.RemoteAddr, dirName, currentPath)
	if dirName == "" {
		return "", "Directory name cannot be empty."
	}
	if strings.ContainsAny(dirName, `/\:*?"<>|`) {
		return "", "Invalid directory name."
	}
	err := os.Mkdir(filepath.Join(currentPath, dirName), os.ModePerm)
	if err != nil {
		return "", "Failed to create directory."
	}
	return fmt.Sprintf("Directory '%s' created.", dirName), ""
}

func handleCreateTxt(w http.ResponseWriter, r *http.Request, currentPath string) {
	fileName := r.FormValue("filename")
	currentRelPath := r.FormValue("path")
	appLogger.Printf("CREATETXT by %s: creating file '%s' in '%s'", r.RemoteAddr, fileName, currentPath)

	var redirectURL string
	if fileName == "" || strings.ContainsAny(fileName, `/\:*?"<>|`) {
		errorMsg := "Invalid or empty file name."
		redirectURL = fmt.Sprintf("/?path=%s&err=%s", url.QueryEscape(currentRelPath), url.QueryEscape(errorMsg))
	} else {
		filePath := filepath.Join(currentPath, fileName)
		if _, err := os.Stat(filePath); err == nil {
			errorMsg := fmt.Sprintf("File '%s' already exists.", fileName)
			redirectURL = fmt.Sprintf("/?path=%s&err=%s", url.QueryEscape(currentRelPath), url.QueryEscape(errorMsg))
		} else if file, err := os.Create(filePath); err != nil {
			errorMsg := "Failed to create file."
			redirectURL = fmt.Sprintf("/?path=%s&err=%s", url.QueryEscape(currentRelPath), url.QueryEscape(errorMsg))
		} else {
			file.Close()
			redirectURL = fmt.Sprintf("/edit?file=%s", url.QueryEscape(filepath.ToSlash(filepath.Join(currentRelPath, fileName))))
		}
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func handleDelete(r *http.Request) (string, string) {
	itemPath := r.FormValue("item")
	appLogger.Printf("DELETE by %s: deleting '%s'", r.RemoteAddr, itemPath)
	safePath, err := getSafePath(itemPath)
	if err != nil {
		return "", "Invalid path for deletion."
	}
	itemName := filepath.Base(itemPath)
	if err = os.RemoveAll(safePath); err != nil {
		return "", fmt.Sprintf("Failed to delete '%s'.", itemName)
	}
	return fmt.Sprintf("'%s' deleted.", itemName), ""
}

func handleRename(r *http.Request) (string, string) {
	oldPathRaw := r.FormValue("old_path")
	newName := r.FormValue("new_name")
	appLogger.Printf("RENAME by %s: renaming '%s' to '%s'", r.RemoteAddr, oldPathRaw, newName)
	if newName == "" {
		return "", "New name cannot be empty."
	}
	if strings.ContainsAny(newName, `/\:*?"<>|`) {
		return "", "Invalid new name."
	}

	oldSafePath, err := getSafePath(oldPathRaw)
	if err != nil {
		return "", "Invalid old path."
	}
	newSafePath := filepath.Join(filepath.Dir(oldSafePath), newName)
	if err := os.Rename(oldSafePath, newSafePath); err != nil {
		return "", fmt.Sprintf("Failed to rename: %v", err)
	}
	return fmt.Sprintf("Renamed '%s' to '%s'.", filepath.Base(oldPathRaw), newName), ""
}

func getSafePath(relativePath string) (string, error) {
	absPath := filepath.Join(*rootPath, relativePath)
	cleanedPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", err
	}
	rootAbs, _ := filepath.Abs(*rootPath)
	if !strings.HasPrefix(cleanedPath, rootAbs) {
		return "", fmt.Errorf("invalid path: access denied")
	}
	return cleanedPath, nil
}

func formatFileSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
