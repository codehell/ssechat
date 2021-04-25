package main

import (
	"encoding/json"
	"fmt"
	"github.com/codehell.net/chat/sse"
	"github.com/gorilla/sessions"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
)

type Login struct {
	ClientId   string `json:"clientId"`
	Credential string `json:"credential"`
	SelectBy   string `json:"select_by"`
}

type Message struct {
	Username string `json:"username"`
	Content string `json:"content"`
}

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

func main() {
	tpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	ms := sse.NewMySSE()
	http.Handle("/static", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := tpl.ExecuteTemplate(w, "index.html", nil)
		if err != nil {
			log.Println(err)
			return
		}
	})
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		err := tpl.ExecuteTemplate(w, "chat.html", nil)
		if err != nil {
			log.Println(err)
			return
		}
	})
	http.Handle("/my-sse", ms)
	http.HandleFunc("/fetch/chat", chat(ms))
	http.HandleFunc("/fetch/login", login)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func chat(ms *sse.MySSE) http.HandlerFunc {
	return func(_ http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "codehellchat")
		username, ok := session.Values["username"]
		if !ok {
			log.Println("username is not set")
			return
		}
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
		usernameText, ok := username.(string)
		if !ok {
			log.Println("error: username is not in text format")
			return
		}
		messageToSent := Message{
			Content: content,
			Username: usernameText,
		}
		jsonMessage, err := json.Marshal(messageToSent)
		if err != nil {
			log.Println("error: can not marshal message")
			return
		}
		logText := fmt.Sprintf("message: \"%s\" sent from %s\n", content, username)
		log.Println(logText)
		if ok {
			ms.SendMessage(string(jsonMessage))
		} else {
			log.Println("warning: message not sent cuz not content")
		}
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	var login Login
	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		log.Println(err)
		return
	}
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + login.Credential)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Println("invalid status response " + resp.Status + " for credential: " + login.Credential)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var accountValues map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&accountValues); err != nil {
		log.Println(err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)
	username, ok := accountValues["given_name"]
	if !ok {
		log.Println("error: username is not present in google user data")
		return
	}
	session, _ := store.Get(r, "codehellchat")
	session.Values["username"] = username
	if err := session.Save(r, w); err != nil {
		log.Println(err)
		return
	}
}
