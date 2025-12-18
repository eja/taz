// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	roomClients = make(map[*websocket.Conn]bool)
	roomMutex   = sync.Mutex{}
)

func mediaRoomHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		appLogger.Printf("WS upgrade error: %v", err)
		return
	}
	defer ws.Close()

	roomMutex.Lock()
	roomClients[ws] = true
	roomMutex.Unlock()

	appLogger.Printf("Room client connected: %s", ws.RemoteAddr())

	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}

		roomMutex.Lock()
		for client := range roomClients {
			if client != ws {
				if err := client.WriteMessage(msgType, msg); err != nil {
					client.Close()
					delete(roomClients, client)
				}
			}
		}
		roomMutex.Unlock()
	}

	roomMutex.Lock()
	delete(roomClients, ws)
	roomMutex.Unlock()
	appLogger.Printf("Room client disconnected: %s", ws.RemoteAddr())
}

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

	if strings.HasSuffix(strings.ToLower(absPath), ".apk") {
		w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	}

	http.ServeFile(w, r, absPath)
}

func bbsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		requireAuth(handleBBSPost, true)(w, r)
		return
	}
	if options.BBSPath == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

	var messages []BBSMessage

	file, err := os.Open(options.BBSPath)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		idCounter := 1
		for scanner.Scan() {
			var msg BBSMessage
			if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
				msg.ID = idCounter
				messages = append(messages, msg)
				idCounter++
			}
		}
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	const messagesPerPage = 10
	totalMessages := len(messages)
	totalPages := (totalMessages + messagesPerPage - 1) / messagesPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * messagesPerPage
	end := start + messagesPerPage
	if end > totalMessages {
		end = totalMessages
	}

	var displayedMessages []BBSMessage
	if start < totalMessages {
		displayedMessages = messages[start:end]
	}

	pages := make([]int, totalPages)
	for i := 0; i < totalPages; i++ {
		pages[i] = i + 1
	}

	data := BBSPageData{
		Title:       "BBS Messages",
		Messages:    displayedMessages,
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

	messageContent := strings.TrimSpace(r.FormValue("message"))
	if messageContent == "" {
		http.Redirect(w, r, "/bbs", http.StatusSeeOther)
		return
	}

	msg := BBSMessage{
		Message:   messageContent,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		appLogger.Printf("Failed to marshal BBS message: %v", err)
		http.Redirect(w, r, "/bbs", http.StatusSeeOther)
		return
	}

	f, err := os.OpenFile(options.BBSPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		appLogger.Printf("Failed to open BBS file: %v", err)
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := f.Write(msgBytes); err != nil {
		appLogger.Printf("Failed to write BBS message: %v", err)
	} else {
		f.WriteString("\n")
		appLogger.Printf("BBS message posted by %s", r.RemoteAddr)
	}

	http.Redirect(w, r, "/bbs", http.StatusSeeOther)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := map[string]interface{}{
		"name":      appLabel,
		"version":   appVersion,
		"time":      time.Now().Unix(),
		"ips":       getServingIPs(),
		"port":      options.WebPort,
		"uptime":    int(time.Since(uptime).Seconds()),
		"discovery": getDiscoveredPeers(),
	}

	json.NewEncoder(w).Encode(status)
}
