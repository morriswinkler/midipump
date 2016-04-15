package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"time"

	"github.com/janberktold/sse"
)

func serverEvents(msgChan chan note) {

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
		go pumpSeq(midiNoteChan)

	})
	http.HandleFunc("/pumpAllStart", func(w http.ResponseWriter, r *http.Request) {
		go pumpAllStart(midiNoteChan)

	})
	http.HandleFunc("/pumpAllStop", func(w http.ResponseWriter, r *http.Request) {
		go pumpAllStop(midiNoteChan)

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, ("/root/go/src/git.laydrop.com/m.winkler/midipump/static" + r.URL.Path))
	})

	http.HandleFunc("/upload", upload)

	http.ListenAndServe(":80", nil)

}

// upload logic
func upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Fprintf(w, "%v", handler.Header)
	f, err := os.OpenFile("/root/go/src/git.laydrop.com/m.winkler/midipump/csv/upload.csv", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	err = pumps.readCsvFile("/root/go/src/git.laydrop.com/m.winkler/midipump/csv/upload.csv")
	if err != nil {
		Error.Printf("error reading csv file: %s", err)
	}

}
