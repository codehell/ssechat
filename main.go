package main

import (
	"encoding/json"
	"github.com/codehell.net/chat/sse"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type Login struct {
	ClientId string `json:"clientId"`
	Credential string `json:"credential"`
	SelectBy string `json:"select_by"`
}

func main() {
	ms := sse.NewMySSE()
	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.Handle("/my-sse", ms)

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		message := make(map[string]string)
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			log.Println(err)
			return
		}
		content, ok := message["content"]
		if ok {
			ms.Clients.RLock()
			for clientID := range ms.Clients.Clients {
				log.Println("message sent to client", clientID)
				ms.MessageChannel <- content
			}
			ms.Clients.RUnlock()
		}
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		var login Login
		err := json.NewDecoder(r.Body).Decode(&login)
		if err != nil {
			log.Println(err)
			return
		}
		resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + login.Credential)

		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != 200 {
			log.Println("invalid status response " +resp.Status+ " for credential: " + login.Credential)
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println(err)
			}
		}(resp.Body)

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(body))
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
