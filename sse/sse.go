package sse

import (
	"encoding/json"
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

type Message struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

func NewMySSE() *MySSE {
	ms := &MySSE{
		MessageChannel: make(chan string),
		Clients: ClientsLocker{
			Clients: make(map[string]bool),
		},
	}
	ticker := time.NewTicker(time.Second * 50)
	go func(t *time.Ticker) {
		message := Message{
			Source:  "heartbeat",
			Content: "heartbeat",
		}
		for range t.C {
			ms.SendMessage(message)
		}
	}(ticker)
	return ms
}

func (s *MySSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
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
		clientIDMessage := make(map[string]string)
		clientIDMessage["source"] = "heartbeat"
		clientIDMessage["content"] = clientID
		clientIDMessageJson, err := json.Marshal(clientIDMessage)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = fmt.Fprintf(w, "data: %s\n\n", clientIDMessageJson)
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

func (s *MySSE) SendMessage(message Message) {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("error: can not marshal message")
		return
	}
	s.Clients.RLock()
	for clientID := range s.Clients.Clients {
		log.Println("message sent to client", clientID)
		s.MessageChannel <- string(jsonMessage)
	}
	s.Clients.RUnlock()
}
