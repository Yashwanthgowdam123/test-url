package main

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
)

//go:embed web/index.html
var web embed.FS

type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Desc  string `json:"desc"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

const seed = `[
["Portfolio (Old)","https://yashwanthgowdam.in/","Personal portfolio showcasing projects, skills, certifications, and professional profile.","globe","#2563eb"],

["Credit Card Bill Splitter","https://creditbillsplit.yashwanthgowdam.in/","Upload and organize credit card statements, split expenses, and generate payment reports.","creditcard","#16a34a"],

["DevOps Interview Questions","https://devopsqna.yashwanthgowdam.in/","Comprehensive DevOps interview questions covering Linux, Docker, Kubernetes, AWS, Terraform, Jenkins, and more.","book","#dc2626"],

["Markup Reader & Creation","https://md.yashwanthgowdam.in/","Online Markdown editor with live preview for creating and reading Markdown documents.","markdown","#7c3aed"],

["Terraform State File to Excel","https://tf2excel.yashwanthgowdam.in/","Convert Terraform state (.tfstate) files into structured Excel spreadsheets.","table","#0f766e"],

["Learn About Subnets","https://learnsubnet.yashwanthgowdam.in/","Interactive subnetting calculator and learning platform for CIDR and IP networking.","network","#ea580c"],

["Basic Multi-Platform Server Inventory","https://server-inventory.yashwanthgowdam.in/","Lightweight inventory system for managing Linux, Windows, and cloud server details.","server","#1d4ed8"],

["Custom Artifactory","https://artifactory.yashwanthgowdam.in/browser/","Self-hosted artifact repository for storing and managing software packages and binaries.","package","#9333ea"],

["Assignment Portal Demo","https://demo-assignment-portal.yashwanthgowdam.in/#/auth/login","Demo assignment submission and management portal with authentication and workflow features.","graduationcap","#0891b2"],

["Images to PDF","https://images2pdf.yashwanthgowdam.in/","Convert one or multiple images into a single PDF with customizable ordering and layout.","image","#f59e0b"]
]`

var (
	mu       sync.Mutex
	items    []Item
	dataFile = env("DATA_FILE", "/data/urls.json")
	adminPw  = os.Getenv("ADMIN_PASSWORD")
	tokens   = map[string]bool{}
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func rid() string { b := make([]byte, 8); rand.Read(b); return hex.EncodeToString(b) }
func save()       { b, _ := json.Marshal(items); os.WriteFile(dataFile, b, 0o644) }
func load() {
	if b, err := os.ReadFile(dataFile); err == nil && json.Unmarshal(b, &items) == nil {
		return
	}
	var s [][]string
	json.Unmarshal([]byte(seed), &s)
	for _, r := range s {
		items = append(items, Item{rid(), r[0], r[1], r[2], r[3], r[4]})
	}
	save()
}
func authed(r *http.Request) bool {
	c, err := r.Cookie("t")
	mu.Lock()
	defer mu.Unlock()
	return err == nil && tokens[c.Value]
}
func guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authed(r) {
			http.Error(w, "unauthorized", 401)
			return
		}
		h(w, r)
	}
}
func readItem(r *http.Request) (Item, bool) {
	var it Item
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<14)
	if json.NewDecoder(r.Body).Decode(&it) != nil {
		return it, false
	}
	it.Name, it.URL, it.Desc = strings.TrimSpace(it.Name), strings.TrimSpace(it.URL), strings.TrimSpace(it.Desc)
	ok := it.Name != "" && it.Desc != "" && len(it.Desc) <= 120 &&
		(strings.HasPrefix(it.URL, "http://") || strings.HasPrefix(it.URL, "https://"))
	return it, ok
}
func js(w http.ResponseWriter, v any) { w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(v) }

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-health" {
		r, err := http.Get("http://127.0.0.1:8080/api/urls")
		if err != nil || r.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}
	load()
	m := http.NewServeMux()
	m.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		b, _ := web.ReadFile("web/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	})
	m.HandleFunc("GET /api/urls", func(w http.ResponseWriter, r *http.Request) { mu.Lock(); defer mu.Unlock(); js(w, items) })
	m.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) { js(w, map[string]bool{"admin": authed(r)}) })
	m.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Password string }
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&b)
		if adminPw == "" || subtle.ConstantTimeCompare([]byte(b.Password), []byte(adminPw)) != 1 {
			http.Error(w, "bad password", 401)
			return
		}
		t := rid() + rid()
		mu.Lock()
		tokens[t] = true
		mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "t", Value: t, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: 86400})
		js(w, map[string]bool{"admin": true})
	})
	m.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("t"); err == nil {
			mu.Lock()
			delete(tokens, c.Value)
			mu.Unlock()
		}
		http.SetCookie(w, &http.Cookie{Name: "t", Path: "/", MaxAge: -1})
		js(w, map[string]bool{"admin": false})
	})
	m.HandleFunc("POST /api/urls", guard(func(w http.ResponseWriter, r *http.Request) {
		it, ok := readItem(r)
		if !ok {
			http.Error(w, "invalid", 400)
			return
		}
		it.ID = rid()
		if it.Color == "" {
			it.Color = "#1d4ed8"
		}
		mu.Lock()
		items = append(items, it)
		save()
		mu.Unlock()
		js(w, it)
	}))
	m.HandleFunc("PUT /api/urls/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		it, ok := readItem(r)
		if !ok {
			http.Error(w, "invalid", 400)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for i := range items {
			if items[i].ID == r.PathValue("id") {
				items[i].Name, items[i].URL, items[i].Desc = it.Name, it.URL, it.Desc
				save()
				js(w, items[i])
				return
			}
		}
		http.NotFound(w, r)
	}))
	m.HandleFunc("DELETE /api/urls/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		for i := range items {
			if items[i].ID == r.PathValue("id") {
				items = append(items[:i], items[i+1:]...)
				save()
				break
			}
		}
		w.WriteHeader(204)
	}))
	http.ListenAndServe(":8080", m)
}
