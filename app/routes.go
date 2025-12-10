// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"io/fs"
	"log"
	"net/http"
)

func setupRoutes() {
	assetsFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		log.Fatalf("Failed to access embedded assets: %v", err)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assetsFS))))

	http.HandleFunc("/", fileManagerHandler)
	http.HandleFunc("/download", downloadHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/bbs", bbsHandler)
	http.HandleFunc("/room", mediaRoomHandler)
}
