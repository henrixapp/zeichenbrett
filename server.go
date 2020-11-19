package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/henrixapp/zeichenbrett/server"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	myhttp := http.NewServeMux()
	myhttp.HandleFunc("/socket", server.SocketReaderCreate)
	myhttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// do NOT do this. (see below)
		http.ServeFile(w, r, "index.html")
	})
	http.ListenAndServe(":8787", myhttp)
}
