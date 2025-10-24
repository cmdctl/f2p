package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Custom HTTP client with appropriate timeouts for file transfers
var httpClient = &http.Client{
	Timeout: 0, // No timeout to allow indefinite file transfers
}

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
		http.HandleFunc("/upload", senderHandler)
		http.HandleFunc("/recv", recvHandler)
		http.HandleFunc("/download", downloadHandler)
		http.HandleFunc("/delete", deleteHandler)
		http.HandleFunc("/", uploadPageHandler)

		// Create server with custom timeouts to handle large file transfers
		server := &http.Server{
			Addr:         ":" + port,
			ReadTimeout:  0,                 // No timeout for reading request body (for large uploads)
			WriteTimeout: 0,                 // No timeout for writing response (for large downloads)
			IdleTimeout:  120 * time.Minute, // Idle timeout to prevent connection leaks
		}

		log.Println("starting server on port:", port)
		log.Fatal(server.ListenAndServe())

	default:
		usage()
	}
}

func usage() {
	fmt.Println("USAGE: f2p <cmd> <options>")
	fmt.Println("COMMANDS:")
	fmt.Println("    server")
	fmt.Println("    send <url> <filepath>")
}
