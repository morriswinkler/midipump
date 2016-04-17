package main

import (
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
	http.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		Info.Println("before home")
		rumbaChan <- "home"
		Info.Println("after home")
	})
	http.HandleFunc("/move", func(w http.ResponseWriter, r *http.Request) {
		rumbaChan <- "move"

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, ("static" + r.URL.Path))
	})

	http.HandleFunc("/upload", upload)

	if emulate {
		http.ListenAndServe(":8080", nil)
	} else {
		http.ListenAndServe(":80", nil)
	}
}

// upload logic
func upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	Info.Println(r)
	file, _, err := r.FormFile("uploadfile")
	if err != nil {
		Error.Printf("unable to read uploadfile: %s", err)
		return
	}
	defer file.Close()

	// get filepath
	fileName := getBasePath("csv/upload.csv")

	// delete old file
	err = os.Remove(fileName)
	if err != nil {
		Error.Printf("unable to remove uploadfile: %s", err)
	}

	// write new file
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		Error.Printf("unable to crate uploadfile: %s", err)
		return
	}

	io.Copy(f, file)
	f.Close()

	// load CSV File
	err = loadCsv()
	if err != nil {
		Error.Printf("unable to load csv file: %s", err)
	}

}
