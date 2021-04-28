package main

import (
	"encoding/json"
	"fmt"
	"github.com/codehell.net/chat/sse"
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	tpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	ms := sse.NewMySSE()
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static", fileServer))

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
	http.HandleFunc("/fetch/login", sse.Login)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s\n", port)
	}
	log.Printf("Listenint on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func chat(ms *sse.MySSE) http.HandlerFunc {
	return func(_ http.ResponseWriter, r *http.Request) {
		session, _ := sse.Store.Get(r, "codehellchat")
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
		messageToSent := sse.Message{
			Content: content,
			Source:  usernameText,
		}
		logText := fmt.Sprintf("message: \"%s\" sent from %s\n", content, username)
		log.Println(logText)
		if ok {
			ms.SendMessage(messageToSent)
		} else {
			log.Println("warning: message not sent cuz not content")
		}
	}
}
