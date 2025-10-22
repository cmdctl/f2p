package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var peerMap sync.Map

func main() {
	cmd := os.Args[1]
	switch cmd {
	case "help":
		usage()

	case "send":
		relayURL := os.Args[2]
		filePath := os.Args[3]
		sendFile(relayURL, filePath)

	case "server":
		port := os.Args[2]
		if port == "" {
			port = "9000"
		}
		http.HandleFunc("/send", senderHandler)
		http.HandleFunc("/recv", recvHandler)
		log.Println("starting server on port: ", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))

	default:
		usage()
	}
}

func recvHandler(w http.ResponseWriter, r *http.Request) {
	senderID := r.URL.Query().Get("id")
	if senderID == "" {
		w.WriteHeader(404)
		fmt.Fprint(w, "sender not found")
		return
	}
	v, ok := peerMap.Load(senderID)
	if !ok {
		w.WriteHeader(404)
		fmt.Fprint(w, "sender id not found")
		return
	}
	peer := v.(Peer)
	peer.ack <- struct{}{}
	io.Copy(w, peer.data)
	peer.done <- struct{}{}
}

func sendFile(url, filePath string) {
	r, w := io.Pipe()
	m := multipart.NewWriter(w)
	go func() {
		defer w.Close()
		defer m.Close()
		part, err := m.CreateFormFile("senderfile", filepath.Base(filePath))
		if err != nil {
			return
		}
		file, err := os.Open(filePath)
		if err != nil {
			return
		}
		defer file.Close()
		if _, err = io.Copy(part, file); err != nil {
			return
		}
	}()
	resp, err := http.Post(url, m.FormDataContentType(), r)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func usage() {
	fmt.Println("USAGE: p2pshare <cmd> <options>")
	fmt.Println("COMMANDS:")
	fmt.Println("    server <port>")
	fmt.Println("    send <url> <filepath>")
	fmt.Println("    recv <ID>")
}

type Peer struct {
	data io.Reader
	done chan struct{}
	ack  chan struct{}
}

func senderHandler(w http.ResponseWriter, r *http.Request) {
	serverHost := os.Getenv("P2PSHARE_HOST")
	if serverHost == "" {
		log.Println("[WARNING] Environment variable P2PSHARE_HOST not set")
	}
	file, _, err := r.FormFile("senderfile")
	if err != nil {
		http.Error(w, "server did not get the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Make sure the response can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Send headers immediately to start chunked transfer
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	senderID := generateID()
	fmt.Fprintf(w, "ID: %s\n", senderID)
	flusher.Flush()

	doneCh := make(chan struct{})
	ackCh := make(chan struct{})
	peer := Peer{data: file, done: doneCh, ack: ackCh}
	peerMap.Store(senderID, peer)

	fmt.Fprintf(w, "Download link: %s/recv?id=%s\n", serverHost, senderID)
	flusher.Flush()

	fmt.Fprint(w, "Waiting for receiver to connect...\n")
	flusher.Flush()

	<-ackCh
	fmt.Fprint(w, "Receiver connected! Transferring data...\n")
	flusher.Flush()

	<-doneCh
	fmt.Fprint(w, "Done!\n")
	flusher.Flush()
}

func generateID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprint(r.Int63())
}
