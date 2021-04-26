package sse

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"sync"
	"time"
)

type ClientsLocker struct {
	sync.RWMutex
	Clients map[string]bool
}

type MySSE struct {
	MessageChannel chan string
	Clients ClientsLocker
}

func NewMySSE() *MySSE {
	return &MySSE{
		MessageChannel: make(chan string),
		Clients: ClientsLocker{
			Clients: make(map[string]bool),
		},
	}
}

func (s *MySSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Println("a new connection has entered")
		h := w.Header()
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
		s.Clients.Lock()
		s.Clients.Clients[clientID] = true
		s.Clients.Unlock()
		flusher := w.(http.Flusher)
		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(w, "data: clientID %s\n\n", clientID)
		if err != nil {
			log.Println(err)
			return
		}
		flusher.Flush()
		for {
			select {
			case <-r.Context().Done():
				log.Println("Client closed", clientID)
				delete(s.Clients.Clients, clientID)
				return
			case m := <-s.MessageChannel:
				_, err := fmt.Fprintf(w, "data: %s\n\n", m)
				if err != nil {
					log.Println(err)
				}
				flusher.Flush()
			}
		}
	}
}

func (s *MySSE) SendMessage(message string) {
	s.Clients.RLock()
	for clientID := range s.Clients.Clients {
		log.Println("message sent to client", clientID)
		s.MessageChannel <- message
	}
	s.Clients.RUnlock()
}
