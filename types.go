package main

import (
	"net/http"
	"sync"
)

var peerMap sync.Map

type Peer struct {
	w    http.ResponseWriter
	done chan struct{}
}
