package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"
)

var clients map[string]bool
var ch chan string

func main() {
	ms := NewMySSE()
	ch = make(chan string)
	clients = make(map[string]bool)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.Handle("/my-sse", ms)

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		message := make(map[string]string)
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			log.Println(err)
			return
		}
		content, ok := message["content"]
		if ok {
			for clientID, _ := range clients {
				log.Println("message sent to client", clientID)
				ch <- content
			}
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type MySSE struct {
	events []string
}

func NewMySSE() *MySSE {
	return &MySSE{
		events: nil,
	}
}

func (s *MySSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()

	if r.Method == "GET" {
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		clientUUID, err := uuid.NewUUID()
		var clientID string
		if err != nil {
			log.Println(err)
			clientID = time.Now().String()
		} else {
			clientID = clientUUID.String()
		}
		clients[clientID] = true
		flusher := w.(http.Flusher)
		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(w, "data: clientID %s\n\n", clientID)
		if err != nil {
			log.Println(err)
		}
		flusher.Flush()
		for {
			select {
			case <-r.Context().Done():
				log.Println("Client closed", clientID)
				delete(clients, clientID)
				return
			case m := <-ch:
				_, err := fmt.Fprintf(w, "data: %s\n\n", m)
				if err != nil {
					log.Println(err)
				}
				flusher.Flush()
			}
		}
	}
}
