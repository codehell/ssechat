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
	Clients map[string]string
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
			Clients: make(map[string]string),
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
		session, _ := Store.Get(r, "codehellchat")
		email := session.Values["email"]
		if email == nil {
			log.Print("error: user is not authorized")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		clientUUID, err := uuid.NewUUID()
		var clientID string
		if err != nil {
			log.Println(err)
			clientID = time.Now().String()
		} else {
			clientID = clientUUID.String()
		}
		// Send list of clients
		jsonClients, err := json.Marshal(s.Clients.Clients)
		if err != nil {
			log.Println(err)
			return
		}
		jsonClientsString := string(jsonClients)
		flusher := w.(http.Flusher)
		// Send list of clients to new client
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("data: "+jsonClientsString+"\n\n",))
		if err != nil {
			log.Println(err)
			return
		}
		flusher.Flush()
		stringEmail, ok := email.(string)
		if !ok {
			stringEmail = "anonymous"
		}
		// Notify that a new client is online
		noticeNewClient := Message{
			Source: "newClient",
			Content: stringEmail,
		}
		s.SendMessage(noticeNewClient)
		s.Clients.Lock()
		s.Clients.Clients[clientID] = stringEmail
		s.Clients.Unlock()
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
