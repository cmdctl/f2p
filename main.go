package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var peerMap sync.Map

func main() {
	if len(os.Args) == 1 {
		usage()
		return
	}
	cmd := os.Args[1]
	switch cmd {
	case "help":
		usage()

	case "send":
		relayURL := os.Args[2]
		filePath := os.Args[3]
		sendFile(relayURL, filePath)

	case "server":
		port := os.Getenv("P2PSHARE_PORT")
		if port == "" {
			port = "9000"
		}
		http.HandleFunc("/id", idHandler)
		http.HandleFunc("/", senderHandler)
		http.HandleFunc("/recv", recvHandler)
		http.HandleFunc("/download", downloadHandler)
		log.Println("starting server on port:", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))

	default:
		usage()
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	serverHost := os.Getenv("P2PSHARE_HOST")
	if serverHost == "" {
		log.Println("[WARNING] Environment variable P2PSHARE_HOST not set. Using localhost as default")
		serverHost = "http://localhost:9000"
	}
	senderID := r.URL.Query().Get("id")

	downloadLink := fmt.Sprintf("%s/recv?id=%s\n", serverHost, senderID)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>File Ready to Download</title>
<style>
    body {
        font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        background: linear-gradient(135deg, #1e293b, #0f172a);
        color: #f1f5f9;
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100vh;
        margin: 0;
    }
    .card {
        background: #1e293b;
        padding: 2rem;
        border-radius: 1.25rem;
        text-align: center;
        box-shadow: 0 10px 20px rgba(0,0,0,0.3);
        width: 90%%;
        max-width: 400px;
        border: 1px solid #334155;
    }
    h1 {
        font-size: 1.5rem;
        margin-bottom: 1rem;
    }
    p {
        color: #cbd5e1;
        margin-bottom: 1.5rem;
    }
    a.download-btn {
        display: inline-block;
        background: #3b82f6;
        color: white;
        padding: 0.75rem 1.5rem;
        border-radius: 9999px;
        text-decoration: none;
        font-weight: 600;
        transition: background 0.2s ease-in-out;
    }
    a.download-btn:hover {
        background: #2563eb;
    }
</style>
</head>
<body>
    <div class="card">
        <h1>📦 Your file is ready</h1>
        <p>Tap the button below to download it securely.</p>
        <a class="download-btn" href="%s" download>Download</a>
    </div>
</body>
</html>
		`, downloadLink)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func idHandler(w http.ResponseWriter, r *http.Request) {
	serverHost := os.Getenv("P2PSHARE_HOST")
	if serverHost == "" {
		serverHost = "http://localhost:9000"
	}
	id := generateID()
	// prepare the rendezvous channel
	peerCh := make(chan Peer)
	peerMap.Store(id, peerCh)
	fmt.Fprintf(w, "%s/download?id=%s\n", serverHost, id)
}

func usage() {
	fmt.Println("USAGE: f2p <cmd> <options>")
	fmt.Println("COMMANDS:")
	fmt.Println("    server")
	fmt.Println("    send <url> <filepath>")
}

type Peer struct {
	w    http.ResponseWriter
	done chan struct{}
}

func senderHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serverHost := os.Getenv("P2PSHARE_HOST")
	if serverHost == "" {
		log.Println("[WARNING] Environment variable P2PSHARE_HOST not set. Using localhost as default")
		serverHost = "http://localhost:9000"
	}
	senderID := r.URL.Query().Get("id")

	go func() {
		<-ctx.Done()
		log.Printf("[INFO] Sender %s disconnected\n", senderID)
		peerMap.Delete(senderID)
	}()

	fileReader, err := r.MultipartReader()
	if err != nil {
		fmt.Fprint(w, "could not read the request")
		return
	}
	file, err := fileReader.NextPart()
	if err != nil {
		fmt.Fprintf(w, "could not read form part %s", err)
		return
	}

	val, ok := peerMap.Load(senderID)
	if !ok {
		fmt.Fprint(w, "sender not found")
		return
	}

	peerCh := val.(chan Peer)

	peer := <-peerCh

	peer.w.Header().Set("Content-Disposition", "attachment; filename="+file.FileName())
	peer.w.Header().Set("Content-Type", "application/octet-stream")
	peer.w.Header().Set("Content-Transfer-Encoding", "binary")
	io.Copy(peer.w, file)

	close(peer.done)
	fmt.Fprint(w, "File transfer successful!")
}

func recvHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "sender id missing in the query params")
		return
	}

	tunnelCh, ok := peerMap.Load(id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<h1>File exprired or not found. Ask for new download link</h1>")
		return
	}
	tunnel := tunnelCh.(chan Peer)

	donech := make(chan struct{})
	tunnel <- Peer{
		w:    w,
		done: donech,
	}

	<-donech
}

func sendFile(baseURL, filePath string) {
	// 1) obtain id + recv link
	resp, err := http.Get(baseURL + "/id")
	if err != nil {
		log.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	// parse ID from line ".../recv?id=XXXX"
	recvURL := strings.TrimSpace(string(b))
	u, _ := url.Parse(recvURL)
	id := u.Query().Get("id")

	fmt.Println("\nDownload link:", recvURL)

	// 2) stream upload with multipart over /upload?id=...
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		part, err := mw.CreateFormFile("senderfile", filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err = io.Copy(part, file); err != nil {
			pw.CloseWithError(err)
			return
		}
		mw.Close()
	}()

	req, _ := http.NewRequest("POST", baseURL+"/upload?id="+id, pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp2.Body.Close()
	io.Copy(os.Stdout, resp2.Body) // prints "OK"
}

func generateID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprint(r.Int63())
}
