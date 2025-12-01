package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// --- CONFIGURATION ---
const (
	DownloadPath = "./downloads"
	Port         = ":8080"
	YtDlpPath    = "yt-dlp"
	IPCSocket    = "/tmp/pitube.sock"
	BackgroundImage = "background.png"
)

var (
	db               *sql.DB
	playerMutex      sync.Mutex
	currentCmd       *exec.Cmd
	idleCmd          *exec.Cmd
	downloadProgress sync.Map
	localIP          string
)

// --- DATA STRUCTURES ---
type Job struct {
	ID        int    `json:"id"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	Title     string `json:"title"`
	Singer    string `json:"singer"`
	Filename  string `json:"filename"`
	Progress  string `json:"progress"`
	TimeLeft  string `json:"time_left"`
}

type SearchResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	Duration any    `json:"duration_string"`
	URL      string `json:"url"`
	Thumb    string `json:"thumbnail"`
}

type PageData struct {
	Queue   []Job `json:"queue"`
	History []Job `json:"history"`
}

func main() {
	logFile, _ := os.OpenFile("pitube_debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if logFile != nil { log.SetOutput(logFile) }

	localIP = getLocalIP()

	os.MkdirAll(DownloadPath, 0755)
	
	var err error
	db, err = sql.Open("sqlite", "pitube.db")
	if err != nil { log.Fatal("Failed to open DB:", err) }
	defer db.Close()
	
	initDB()      
	syncLibrary() 
	
	go downloadWorker()
	go playerWorker()
	go osdManager()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/data", handle
