// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

func mapHandler(w http.ResponseWriter, r *http.Request) {
	prefix := "/map/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid map URL", http.StatusBadRequest)
		return
	}

	relativePath := strings.TrimPrefix(path, prefix)

	if strings.HasSuffix(relativePath, "/metadata.json") {
		filename := strings.TrimSuffix(relativePath, "/metadata.json")
		mapMetadata(w, filename)
		return
	}

	parts := strings.Split(relativePath, "/")
	n := len(parts)

	if n >= 4 {
		z, errZ := strconv.Atoi(parts[n-3])
		x, errX := strconv.Atoi(parts[n-2])
		y, errY := strconv.Atoi(parts[n-1])

		filename := strings.Join(parts[:n-3], "/")

		if errZ == nil && errX == nil && errY == nil && strings.HasSuffix(filename, ".mbtiles") {
			mapMBTiles(w, filename, z, x, y)
			return
		}
	}

	if strings.HasSuffix(path, ".pmtiles") {
		prefix = "/download/"
	}

	data := map[string]interface{}{
		"File": prefix + relativePath,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "map.html", data)
}

func mapMetadata(w http.ResponseWriter, filename string) {
	if !strings.HasSuffix(filename, ".mbtiles") {
		http.Error(w, "Metadata only supported for mbtiles", http.StatusBadRequest)
		return
	}

	absPath, err := getSafePath(filename)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite", absPath)
	if err != nil {
		http.Error(w, "Failed to open mbtiles file", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, value FROM metadata")
	if err != nil {
		http.Error(w, "Failed to read metadata", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	metadata := make(map[string]interface{})
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}

		var jsonValue interface{}
		if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
			metadata[name] = jsonValue
		} else {
			metadata[name] = value
		}
	}

	for _, key := range []string{"minzoom", "maxzoom"} {
		if val, ok := metadata[key].(string); ok {
			if num, err := strconv.Atoi(val); err == nil {
				metadata[key] = num
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(metadata)
}

func mapMBTiles(w http.ResponseWriter, filename string, z, x, y int) {
	absPath, err := getSafePath(filename)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite", absPath)
	if err != nil {
		http.Error(w, "Failed to open mbtiles file", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	tmsY := (1 << z) - 1 - y

	var tileData []byte
	err = db.QueryRow("SELECT tile_data FROM tiles WHERE zoom_level = ? AND tile_column = ? AND tile_row = ?",
		z, x, tmsY).Scan(&tileData)

	if err == sql.ErrNoRows {
		http.Error(w, "Tile not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(tileData)
}
