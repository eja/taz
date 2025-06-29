// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
		KB = 1024
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

	isAuthenticated := false
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

func handleCreateTxt(w http.ResponseWriter, r *http.Request, currentPath string) {
	fileName := r.FormValue("filename")
	currentRelPath := r.FormValue("path")
	appLogger.Printf("CREATE by %s: creating file '%s' in '%s'", r.RemoteAddr, fileName, currentPath)

	var redirectURL string
	if fileName == "" || strings.ContainsAny(fileName, `/\:*?"<>|`) {
		redirectURL = fmt.Sprintf("/?path=%s&err=%s", url.QueryEscape(currentRelPath), url.QueryEscape("Invalid or empty file name."))
	} else {
		filePath := filepath.Join(currentPath, fileName)
		if _, err := os.Stat(filePath); err == nil {
			redirectURL = fmt.Sprintf("/?path=%s&err=%s", url.QueryEscape(currentRelPath), url.QueryEscape(fmt.Sprintf("File '%s' already exists.", fileName)))
		} else if file, err := os.Create(filePath); err != nil {
			redirectURL = fmt.Sprintf("/?path=%s&err=%s", url.QueryEscape(currentRelPath), url.QueryEscape("Failed to create file."))
		} else {
			file.Close()
			redirectURL = fmt.Sprintf("/edit?file=%s", url.QueryEscape(filepath.ToSlash(filepath.Join(currentRelPath, fileName))))
		}
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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

	var uploaded []string
	for _, f := range files {
		src, err := f.Open()
		if err != nil {
			return "", fmt.Sprintf("Failed to open file '%s'.", f.Filename)
		}
		defer src.Close()

		clean := filepath.Base(f.Filename)
		if strings.ContainsAny(clean, `/\:*?"<>|`) {
			continue
		}
		dstPath := filepath.Join(destPath, clean)
		dst, err := os.Create(dstPath)
		if err != nil {
			return "", fmt.Sprintf("Could not create file '%s'.", clean)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return "", fmt.Sprintf("Failed to save file '%s'.", clean)
		}
		uploaded = append(uploaded, clean)
		appLogger.Printf("UPLOAD by %s: processing file '%s' (size: %d)", r.RemoteAddr, f.Filename, f.Size)
	}

	if len(uploaded) == 0 {
		return "", "No valid files were uploaded."
	}
	return fmt.Sprintf("Uploaded: %s", strings.Join(uploaded, ", ")), ""
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
	if err := os.Mkdir(filepath.Join(currentPath, dirName), os.ModePerm); err != nil {
		return "", "Failed to create directory."
	}
	return fmt.Sprintf("Directory '%s' created.", dirName), ""
}

func handleDelete(r *http.Request) (string, string) {
	itemPath := r.FormValue("item")
	appLogger.Printf("DELETE by %s: deleting '%s'", r.RemoteAddr, itemPath)
	safePath, err := getSafePath(itemPath)
	if err != nil {
		return "", "Invalid path for deletion."
	}
	if err := os.RemoveAll(safePath); err != nil {
		return "", fmt.Sprintf("Failed to delete '%s'.", filepath.Base(itemPath))
	}
	return fmt.Sprintf("'%s' deleted.", filepath.Base(itemPath)), ""
}

func handleRename(r *http.Request) (string, string) {
	oldPath := r.FormValue("old_path")
	newName := r.FormValue("new_name")
	appLogger.Printf("RENAME by %s: renaming '%s' to '%s'", r.RemoteAddr, oldPath, newName)
	if newName == "" {
		return "", "New name cannot be empty."
	}
	if strings.ContainsAny(newName, `/\:*?"<>|`) {
		return "", "Invalid new name."
	}
	oldSafePath, err := getSafePath(oldPath)
	if err != nil {
		return "", "Invalid old path."
	}
	newSafePath := filepath.Join(filepath.Dir(oldSafePath), newName)
	if err := os.Rename(oldSafePath, newSafePath); err != nil {
		return "", fmt.Sprintf("Failed to rename: %v", err)
	}
	return fmt.Sprintf("Renamed '%s' to '%s'.", filepath.Base(oldPath), newName), ""
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
	appLogger.Printf("SAVE by %s: saving file '%s' in '%s'", r.RemoteAddr, relativePath, safePath)
	os.WriteFile(safePath, []byte(content), 0644)
	redirectURL := fmt.Sprintf("/?path=%s&msg=%s",
		url.QueryEscape(filepath.ToSlash(filepath.Dir(relativePath))),
		url.QueryEscape("Saved "+filepath.Base(relativePath)),
	)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
