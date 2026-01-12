// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"encoding/base64"
	"io/fs"
	"log"
	"net/http"

	"github.com/eja/maps"
	"github.com/eja/tabulita"
)

func setupRoutes() {
	webAuth := options.Password
	if webAuth != "" {
		webAuth = base64.StdEncoding.EncodeToString([]byte(":" + options.Password))
	}

	assetsFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		log.Fatalf("Failed to access embedded assets: %v", err)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assetsFS))))

	http.HandleFunc("/", fileManagerHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/download/", downloadHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/bbs", bbsHandler)
	http.HandleFunc("/room", mediaRoomHandler)
	http.Handle("/maps/", maps.New("/maps/", options.RootPath, ""))
	http.Handle("/tabulita/", tabulita.New("/tabulita/", options.RootPath, webAuth))
}
