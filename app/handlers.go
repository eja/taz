// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"crypto/subtle"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	pass := r.FormValue("password")
	if subtle.ConstantTimeCompare([]byte(pass), []byte(options.Password)) == 1 {
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
	relativePath := r.URL.Query().Get("path")
	if relativePath == "" {
		relativePath = "."
	}
	absPath, err := getSafePath(relativePath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		appLogger.Printf("Invalid path access: %s", relativePath)
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
		appLogger.Printf("Invalid path access: %s", relativePath)
		return
	}
	action := r.FormValue("action")
	if action == "createtxt" {
		handleCreateTxt(w, r, absPath)
		return
	}
	var msg, errMsg string
	switch action {
	case "upload":
		msg, errMsg = handleUpload(r, absPath)
	case "mkdir":
		msg, errMsg = handleMkdir(r, absPath)
	case "delete":
		msg, errMsg = handleDelete(r)
	case "rename":
		msg, errMsg = handleRename(r)
	}
	redirect := fmt.Sprintf("/?path=%s&msg=%s&err=%s",
		template.URLQueryEscaper(relativePath),
		template.URLQueryEscaper(msg),
		template.URLQueryEscaper(errMsg),
	)
	http.Redirect(w, r, redirect, http.StatusSeeOther)
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

func bbsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		requireAuth(handleBBSPost, true)(w, r)
		return
	}
	if db == nil {
		return
	}
	handleBBSGet(w, r)
}

func handleBBSGet(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	const messagesPerPage = 10
	offset := (page - 1) * messagesPerPage

	var totalMessages int
	err := db.QueryRow("SELECT COUNT(*) FROM bbs_messages").Scan(&totalMessages)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT id, message, created_at FROM bbs_messages ORDER BY created_at DESC LIMIT ? OFFSET ?", messagesPerPage, offset)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []BBSMessage
	for rows.Next() {
		var msg BBSMessage
		if err := rows.Scan(&msg.ID, &msg.Message, &msg.CreatedAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	totalPages := (totalMessages + messagesPerPage - 1) / messagesPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	pages := make([]int, totalPages)
	for i := 0; i < totalPages; i++ {
		pages[i] = i + 1
	}

	data := BBSPageData{
		Title:       "BBS Messages",
		Messages:    messages,
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrevious: page > 1,
		HasNext:     page < totalPages,
		Pages:       pages,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "bbs.html", data)
}

func handleBBSPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusInternalServerError)
		return
	}

	message := strings.TrimSpace(r.FormValue("message"))
	if message == "" {
		http.Redirect(w, r, "/bbs", http.StatusSeeOther)
		return
	}

	_, err := db.Exec("INSERT INTO bbs_messages (message) VALUES (?)", message)
	if err != nil {
		appLogger.Printf("Failed to save BBS message: %v", err)
	} else {
		appLogger.Printf("BBS message posted by %s", r.RemoteAddr)
	}

	http.Redirect(w, r, "/bbs", http.StatusSeeOther)
}
