package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
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
	DownloadPath    = "./downloads"
	Port            = ":8080"
	YtDlpPath       = "yt-dlp"
	IPCSocket       = "/tmp/pitube.sock"
	BackgroundImage = "background.png"
	LogFile         = "pitube_debug.log"
	MaxLogSize      = 1 * 1024 * 1024 // 1MB
)

// Massive API List for redundancy
var InvidiousInstances = []string{
	"https://invidious.jing.rocks/api/v1/search",
	"https://invidious.nerdvpn.de/api/v1/search",
	"https://inv.zzls.xyz/api/v1/search",
	"https://invidious.io.lol/api/v1/search",
	"https://invidious.private.coffee/api/v1/search",
	"https://iv.ggtyler.dev/api/v1/search",
	"https://invidious.fdn.fr/api/v1/search",
	"https://invidious.perennialteks.com/api/v1/search",
	"https://yt.artemislena.eu/api/v1/search",
	"https://invidious.projectsegfau.lt/api/v1/search",
}

var (
	db               *sql.DB
	playerMutex      sync.Mutex
	currentCmd       *exec.Cmd
	idleCmd          *exec.Cmd
	downloadProgress sync.Map
	localIP          string
)

// --- LOG ROTATOR ---
type LogRotator struct {
	mu sync.Mutex
}

func (l *LogRotator) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	info, err := os.Stat(LogFile)
	if err == nil && info.Size() >= MaxLogSize {
		os.Remove(LogFile + ".old")
		os.Rename(LogFile, LogFile+".old")
	}
	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil { return 0, err }
	defer f.Close()
	return f.Write(p)
}

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

type InvidiousItem struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	VideoID    string `json:"videoId"`
	Author     string `json:"author"`
	Length     int    `json:"lengthSeconds"`
	Thumbnails []struct {
		URL string `json:"url"`
	} `json:"videoThumbnails"`
}

type SearchResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	Duration string `json:"duration_string"`
	URL      string `json:"url"`
	Thumb    string `json:"thumbnail"`
}

type PageData struct {
	Queue   []Job `json:"queue"`
	History []Job `json:"history"`
}

func main() {
	log.SetOutput(&LogRotator{})
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
	http.HandleFunc("/api/data", handleAPIData)
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/retry", handleRetry)
	http.HandleFunc("/api/update_ytdlp", handleUpdateYTDLP)
	http.HandleFunc("/api/shutdown", handleShutdown)
	http.HandleFunc("/add", handleAdd)
	http.HandleFunc("/skip", handleSkip)
	http.HandleFunc("/delete", handleDelete)

	log.Println("Started PiTube Karaoke on " + Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}

func initDB() {
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		log.Println("Warning: Could not enable WAL mode:", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, url TEXT, status TEXT, title TEXT, singer TEXT, filename TEXT, created_at DATETIME);`)
	log.Println("Clearing previous queue on boot...")
	db.Exec("UPDATE jobs SET status = 'played' WHERE status IN ('pending', 'downloading', 'ready', 'playing')")
}

func syncLibrary() {
	log.Println("Scanning download folder...")
	extensions := []string{"*.mp4", "*.webm", "*.mkv"}
	for _, ext := range extensions {
		files, _ := filepath.Glob(filepath.Join(DownloadPath, ext))
		for _, path := range files {
			filename := filepath.Base(path)
			title := strings.TrimSuffix(filename, filepath.Ext(filename))
			var id int
			err := db.QueryRow("SELECT id FROM jobs WHERE filename = ?", filename).Scan(&id)
			if err == sql.ErrNoRows {
				log.Printf("Indexing orphan file: %s", filename)
				db.Exec("INSERT INTO jobs (url, status, title, singer, filename, created_at) VALUES (?, ?, ?, ?, ?, ?)", "local_file", "played", title, "System", filename, time.Now())
			}
		}
	}
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil { return "Unknown" }
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "Unknown"
}

func getMPVRemaining() string {
	c, err := net.DialTimeout("unix", IPCSocket, 200*time.Millisecond)
	if err != nil { return "" }
	defer c.Close()
	c.SetDeadline(time.Now().Add(200 * time.Millisecond))
	c.Write([]byte(`{"command": ["get_property", "time-remaining"]}` + "\n"))
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil { return "" }
	var resp struct { Data float64 `json:"data"` }
	if json.Unmarshal(buf[:n], &resp) != nil || resp.Data <= 0 { return "" }
	m := int(resp.Data) / 60
	s := int(resp.Data) % 60
	return fmt.Sprintf("-%d:%02d", m, s)
}

// --- HANDLERS ---

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil { http.Error(w, "Template error", 500); return }
	tmpl.Execute(w, nil)
}

func handleUpdateYTDLP(w http.ResponseWriter, r *http.Request) {
	log.Println("Updating yt-dlp...")
	script := `
	cd /tmp
	rm -f yt-dlp_linux_armv7l.zip
	wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l.zip
	unzip -o yt-dlp_linux_armv7l.zip -d /opt/yt-dlp/
	chmod +x /opt/yt-dlp/yt-dlp_linux_armv7l
	ln -sf /opt/yt-dlp/yt-dlp_linux_armv7l /usr/local/bin/yt-dlp
	`
	cmd := exec.Command("bash", "-c", script)
	err := cmd.Run()
	if err != nil {
		log.Printf("Update failed: %v", err)
		http.Error(w, "Update failed", 500)
		return
	}
	log.Println("Update successful!")
	w.WriteHeader(http.StatusOK)
}

func handleShutdown(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Shutdown Request. Powering off...")
	cmd := exec.Command("sudo", "poweroff")
	if err := cmd.Start(); err != nil {
		log.Printf("Shutdown failed: %v", err)
		http.Error(w, "Shutdown failed", 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleAPIData(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, url, status, title, singer, filename FROM jobs WHERE status != 'played' ORDER BY id ASC")
	if err != nil { http.Error(w, "DB Error", 500); return }
	defer rows.Close()
	var queue []Job
	timeLeft := getMPVRemaining()
	for rows.Next() {
		var j Job
		rows.Scan(&j.ID, &j.URL, &j.Status, &j.Title, &j.Singer, &j.Filename)
		if j.Status == "downloading" {
			if p, ok := downloadProgress.Load(j.ID); ok { j.Progress = p.(string) }
		}
		if j.Status == "playing" { j.TimeLeft = timeLeft }
		queue = append(queue, j)
	}
	hRows, _ := db.Query("SELECT id, url, status, title, singer, filename FROM jobs WHERE status = 'played' GROUP BY title ORDER BY title ASC")
	var history []Job
	if hRows != nil {
		defer hRows.Close()
		for hRows.Next() {
			var j Job
			hRows.Scan(&j.ID, &j.URL, &j.Status, &j.Title, &j.Singer, &j.Filename)
			history = append(history, j)
		}
	}
    if queue == nil { queue = []Job{} }
    if history == nil { history = []Job{} }
    json.NewEncoder(w).Encode(PageData{Queue: queue, History: history})
}

func handleRetry(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" { db.Exec("UPDATE jobs SET status = 'pending' WHERE id = ?", id) }
	w.WriteHeader(http.StatusOK)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" { return }
	if !strings.Contains(strings.ToLower(query), "karaoke") { query += " karaoke" }
	
	type APIResult struct {
		Items []InvidiousItem
		Error error
	}
	resultChan := make(chan APIResult, 1) 
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) 
	defer cancel()
	var wg sync.WaitGroup
	log.Printf("[SEARCH] Searching for: %s", query)

	for _, apiBase := range InvidiousInstances {
		wg.Add(1)
		go func(urlStr string) {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(ctx, "GET", urlStr+"?q="+url.QueryEscape(query), nil)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil { return }
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				var items []InvidiousItem
				if err := json.NewDecoder(resp.Body).Decode(&items); err == nil && len(items) > 0 {
					select {
					case resultChan <- APIResult{Items: items}:
					default: 
					}
				}
			}
		}(apiBase)
	}

	select {
	case res := <-resultChan:
		log.Println("[SEARCH] Fast API Success!")
		sendSearchResults(w, res.Items)
		return
	case <-ctx.Done():
		log.Println("[SEARCH] API Timeout. Switching to local fallback.")
	}

	searchFlags := []string{
		"--print", "%(id)s<|>%(title)s<|>%(uploader)s<|>%(duration_string)s", 
		"--flat-playlist", "--no-warnings", "--js-runtimes", "node", 
	}
	if _, err := os.Stat("cookies.txt"); err == nil {
		searchFlags = append([]string{"--cookies", "cookies.txt"}, searchFlags...)
	}
	searchFlags = append(searchFlags, "ytsearch5:"+query)

	log.Printf("[CMD] yt-dlp %s", strings.Join(searchFlags, " "))
	cmd := exec.Command(YtDlpPath, searchFlags...)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[SEARCH ERROR] %v", err)
		http.Error(w, "Search failed", 500)
		return
	}

	var results []SearchResult
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "<|>")
		if len(parts) < 4 { continue }
		res := SearchResult{
			ID: parts[0], Title: parts[1], Uploader: parts[2], Duration: parts[3],
			URL: "https://www.youtube.com/watch?v=" + parts[0],
			Thumb: fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", parts[0]),
		}
		results = append(results, res)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func sendSearchResults(w http.ResponseWriter, items []InvidiousItem) {
	var results []SearchResult
	for _, item := range items {
		if item.Type != "video" { continue }
		res := SearchResult{
			ID: item.VideoID, Title: item.Title, Uploader: item.Author,
			URL: "https://www.youtube.com/watch?v=" + item.VideoID,
			Duration: fmt.Sprintf("%d:%02d", item.Length/60, item.Length%60),
			Thumb: fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", item.VideoID),
		}
		if len(item.Thumbnails) > 0 { res.Thumb = item.Thumbnails[0].URL }
		results = append(results, res)
		if len(results) >= 10 { break }
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		url := r.FormValue("url")
		singer := r.FormValue("singer")
		title := r.FormValue("title") 
		if singer == "" { singer = "Mystery Guest" }
		if title == "" { title = url }
		var existingFilename string
		err := db.QueryRow("SELECT filename FROM jobs WHERE url = ? AND filename != '' ORDER BY id DESC LIMIT 1", url).Scan(&existingFilename)
		if err == nil && existingFilename != "" {
			if _, statErr := os.Stat(filepath.Join(DownloadPath, existingFilename)); statErr == nil {
				log.Printf("Instant add for %s (from cache)", title)
				db.Exec("INSERT INTO jobs (url, status, title, singer, filename, created_at) VALUES (?, ?, ?, ?, ?, ?)", url, "ready", title, singer, existingFilename, time.Now())
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		db.Exec("INSERT INTO jobs (url, status, title, singer, created_at) VALUES (?, ?, ?, ?, ?)", url, "pending", title, singer, time.Now())
	}
	w.WriteHeader(http.StatusOK)
}

func handleSkip(w http.ResponseWriter, r *http.Request) {
	playerMutex.Lock()
	if currentCmd != nil && currentCmd.Process != nil { currentCmd.Process.Kill() }
	playerMutex.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	db.Exec("DELETE FROM jobs WHERE id = ?", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func downloadWorker() {
	progressRegex := regexp.MustCompile(`\s*(\d{1,3}(?:\.\d+)?)%`)
	for {
		var id int
		var url string
		err := db.QueryRow("SELECT id, url FROM jobs WHERE status = 'pending' LIMIT 1").Scan(&id, &url)
		if err != nil { time.Sleep(2 * time.Second); continue }

		db.Exec("UPDATE jobs SET status = 'downloading' WHERE id = ?", id)
		outTmpl := filepath.Join(DownloadPath, "%(title)s.%(ext)s")

		formatFlags := []string{
			"--newline", "--no-playlist", 
			"-f", "best[height<=480]/bestvideo[height<=480]+bestaudio/best", 
			"--js-runtimes", "node",
		}
		if _, err := os.Stat("cookies.txt"); err == nil {
			formatFlags = append(formatFlags, "--cookies", "cookies.txt")
		}

		checkArgs := append(formatFlags, "--get-filename", "-o", outTmpl, url)
		outInfo, _ := exec.Command(YtDlpPath, checkArgs...).Output()
		predictedPath := strings.TrimSpace(string(outInfo))
		if predictedPath != "" {
			if _, err := os.Stat(predictedPath); err == nil {
				log.Printf("Job %d: File exists.", id)
				title := strings.TrimSuffix(filepath.Base(predictedPath), filepath.Ext(predictedPath))
				db.Exec("UPDATE jobs SET status = 'ready', filename = ?, title = ? WHERE id = ?", filepath.Base(predictedPath), title, id)
				continue 
			}
		}

		runArgs := append(formatFlags, "-o", outTmpl, url)
		log.Printf("[CMD] Job %d: yt-dlp %s", id, strings.Join(runArgs, " "))
		cmd := exec.Command(YtDlpPath, runArgs...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Job %d Pipe Error: %v", id, err)
			db.Exec("UPDATE jobs SET status = 'failed' WHERE id = ?", id)
			continue
		}
		if err := cmd.Start(); err != nil {
			log.Printf("Job %d Start Error: %v", id, err)
			db.Exec("UPDATE jobs SET status = 'failed' WHERE id = ?", id)
			continue
		}
		scanner := bufio.NewScanner(stdout)
		isFinalizing := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "MiB/s") || strings.Contains(line, "ERROR") || strings.Contains(line, "WARNING") {
				log.Printf("[DL %d] %s", id, line)
			}
			if isFinalizing { continue }
			if strings.Contains(line, "[Merger]") || strings.Contains(line, "Merging formats") || strings.Contains(line, "100%") {
				isFinalizing = true
				downloadProgress.Store(id, "Finalizing")
				continue
			} 
			if strings.Contains(line, "[download]") {
				matches := progressRegex.FindStringSubmatch(line)
				if len(matches) > 1 { downloadProgress.Store(id, matches[1] + "%") }
			}
		}
		if err := cmd.Wait(); err != nil {
			log.Printf("Job %d Download Failed: %v", id, err)
			log.Printf("--- STDERR ---\n%s\n--- END STDERR ---", stderr.String())
			db.Exec("UPDATE jobs SET status = 'failed' WHERE id = ?", id)
			downloadProgress.Delete(id)
			continue
		}
		downloadProgress.Delete(id)

		fullPath := predictedPath
		title := strings.TrimSuffix(filepath.Base(fullPath), filepath.Ext(fullPath))
		if _, err := os.Stat(fullPath); err == nil {
			db.Exec("UPDATE jobs SET status = 'ready', filename = ?, title = ? WHERE id = ?", filepath.Base(fullPath), title, id)
			log.Printf("Job %d Success: %s", id, title)
		} else {
			globPattern := filepath.Join(DownloadPath, title + ".*")
			matches, _ := filepath.Glob(globPattern)
			if len(matches) > 0 {
				log.Printf("Job %d: Recovered: %s", id, matches[0])
				db.Exec("UPDATE jobs SET status = 'ready', filename = ?, title = ? WHERE id = ?", filepath.Base(matches[0]), title, id)
			} else {
				log.Printf("Job %d: File missing: %s", id, fullPath)
				db.Exec("UPDATE jobs SET status = 'failed' WHERE id = ?", id)
			}
		}
	}
}

func playerWorker() {
	os.Remove(IPCSocket)
	for {
		var id int
		var f, title string
		err := db.QueryRow("SELECT id, filename, title FROM jobs WHERE status = 'ready' ORDER BY id ASC LIMIT 1").Scan(&id, &f, &title)
		
		if err != nil { 
			// No song? Just wait. The wallpaper is already there.
			time.Sleep(1 * time.Second) 
			continue 
		}

		log.Printf("Attempting to play: %s", title)
		db.Exec("UPDATE jobs SET status = 'playing' WHERE id = ?", id)
		
		fullPath := filepath.Join(DownloadPath, f)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			log.Printf("Error: File not found on disk: %s", fullPath)
			db.Exec("UPDATE jobs SET status = 'failed' WHERE id = ?", id)
			continue
		}

		cmd := exec.Command("mpv", 
			"--fs", "--ontop", 
			"--input-ipc-server="+IPCSocket, 
			"--osd-align-y=bottom", 
			"--osd-align-x=right", 
			"--osd-margin-y=20", "--osd-margin-x=20",
			"--osd-font-size=25", 
			"--osd-color=#FFFF00", 
			"--osd-level=3",
			"--profile=sw-fast",  
			"--vo=x11",           
			"--framedrop=vo",     
			"--video-sync=desync", 
			fullPath)
		cmd.Env = append(os.Environ(), "DISPLAY=:0")
		
		playerMutex.Lock()
		currentCmd = cmd
		playerMutex.Unlock()
		
		err = cmd.Run()
		if err != nil { log.Printf("MPV exited with error: %v", err) }

		playerMutex.Lock()
		currentCmd = nil
		playerMutex.Unlock()
		
		db.Exec("UPDATE jobs SET status = 'played' WHERE id = ?", id)
		
		log.Println("Song finished. Pausing for 5s (Show QR)...")
		time.Sleep(5 * time.Second)
	}
}

func osdManager() {
	for {
		time.Sleep(5 * time.Second)
		if _, err := os.Stat(IPCSocket); os.IsNotExist(err) { continue }
		
		var s, t string
		err := db.QueryRow("SELECT singer, title FROM jobs WHERE status = 'ready' LIMIT 1").Scan(&s, &t)
		
		// FIX: Updated Text
		osdText := fmt.Sprintf("Join: http://%s:%s", localIP, "8080")
		if err == nil {
			osdText += fmt.Sprintf("                                  UP NEXT: %s (%s)", s, t)
		}

		c, _ := net.DialTimeout("unix", IPCSocket, 200*time.Millisecond)
		if c == nil { continue }
		c.SetDeadline(time.Now().Add(200*time.Millisecond))
		cmd := fmt.Sprintf(`{ "command": ["show-text", "%s", 6000, 3] }` + "\n", osdText)
		c.Write([]byte(cmd))
		c.Close()
	}
}
