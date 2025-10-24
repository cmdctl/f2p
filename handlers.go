package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
)

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	senderID := r.URL.Query().Get("id")
	peerMap.Delete(senderID)
	w.Write([]byte("OK"))
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
		`, html.EscapeString(downloadLink))

	w.Header().Set("Content-Type", "text/html")
	_, ok := peerMap.Load(senderID)
	if !ok {
		fmt.Fprint(w, "<h1>File exprired or not found. Ask for another download link</h1>")
		return
	}
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
