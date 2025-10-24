package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

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
		log.Println("starting server on port:", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))

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

