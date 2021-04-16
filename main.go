package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

var clients []chan string

func main() {
	ms := NewMySSE()
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
			for _, client := range clients {
				log.Println("message sent to client", client)
				client <- content
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
		client := make(chan string)
		clients = append(clients, client)
		s.events = append(s.events, r.Header.Get("Last-Event-ID"))
		log.Println(s.events[len(s.events)-1])
		flusher := w.(http.Flusher)
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		for {
			select {
			case <-r.Context().Done():
				log.Println("Client closed")
				return
			case m := <-client:
				_, err := fmt.Fprintf(w, "data: %s\n\n", m)
				if err != nil {
					log.Println(err)
				}
				flusher.Flush()
				time.Sleep(time.Second * 1)
			}
		}
	}
}
