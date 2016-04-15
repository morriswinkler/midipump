package main

import (
	"net/http"

	"time"

	"github.com/janberktold/sse"
)

func serverEvents(msgChan chan *note) {

	http.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
		// get a SSE connection from the HTTP request
		// in a real world situation, you should look for the error (second return value)
		conn, _ := sse.Upgrade(w, r)

		for {
			// keep this goroutine alive to keep the connection

			// get a message from some channel
			// blocks until it recieves a messages and then instantly sends it to the client
			//msg := <-msgChan
			//Info.Println(msg)

			//for i, note := ranges midiNotes {
			for i := range pumps {
				conn.WriteJson(pumps[i])

			}
			time.Sleep(time.Second * 10)

		}
	})

	http.HandleFunc("/pump", func(w http.ResponseWriter, r *http.Request) {
		go pumpAll(midiNoteChan)

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, ("./static" + r.URL.Path))
	})

	http.ListenAndServe(":8080", nil)

}
