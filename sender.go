package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

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

	peer.w.Header().Set("Content-Disposition", "attachment; filename="+file.FileName())
	peer.w.Header().Set("Content-Type", "application/octet-stream")
	peer.w.Header().Set("Content-Transfer-Encoding", "binary")
	io.Copy(peer.w, file)

	close(peer.done)
	peerMap.Delete(senderID)
	log.Printf("Remove sender id: %s\n", senderID)
	fmt.Fprint(w, "File transfer successful!")
}

func sendFile(baseURL, filePath string) {
	// 1) obtain id + recv link
	req, err := http.NewRequest("GET", baseURL+"/id", nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := httpClient.Do(req)
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

	req, _ = http.NewRequest("POST", baseURL+"/upload?id="+id, pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp2, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp2.Body.Close()
	io.Copy(os.Stdout, resp2.Body) // prints "OK"
}
