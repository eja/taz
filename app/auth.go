// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
)

func getPasswordHash() string {
	if options.Password == "" {
		return ""
	}
	hasher := sha256.New()
	hasher.Write([]byte(options.Password))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func requireAuth(next http.HandlerFunc, requireWrite bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if options.Password == "" {
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
	if options.Password == "" {
		return false
	}
	expectedToken := getPasswordHash()
	return subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1
}
