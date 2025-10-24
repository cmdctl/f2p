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
		log.Println("starting server on port:", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))

	default:
		usage()
	}
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
	fmt.Fprintf(w, "%s/recv?id=%s\n", serverHost, id)
}

func usage() {
	fmt.Println("USAGE: f2p <cmd> <options>")
	fmt.Println("COMMANDS:")
	fmt.Println("    server")
	fmt.Println("    send <url> <filepath>")
}

type Peer struct {
	w    io.Writer
	done chan struct{}
}

func senderHandler(w http.ResponseWriter, r *http.Request) {
	serverHost := os.Getenv("P2PSHARE_HOST")
	if serverHost == "" {
		log.Println("[WARNING] Environment variable P2PSHARE_HOST not set. Using localhost as default")
		serverHost = "http://localhost:9000"
	}
	senderID := r.URL.Query().Get("id")

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
		fmt.Fprint(w, "sender not found")
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
