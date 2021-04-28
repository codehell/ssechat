package sse

import (
	"encoding/json"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
	"os"
)

type LoginFields struct {
	ClientId   string `json:"clientId"`
	Credential string `json:"credential"`
	SelectBy   string `json:"select_by"`
}

var Store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	var login LoginFields
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
	err = resp.Body.Close()
	if err != nil {
		log.Println(err)
		return
	}
	username, ok := accountValues["given_name"]
	if !ok {
		log.Println("error: username is not present in google user data")
		return
	}
	emailVerified, ok := accountValues["email_verified"]
	if !ok {
		log.Println("error: email is not verifier in google user data")
		return
	}
	if emailVerified != "true" {
		log.Println("error: email is not verifier in google user data")
		return
	}
	email, ok := accountValues["email"]
	if !ok {
		log.Println("error: email is not present in google user data")
		return
	}
	session, _ := Store.Get(r, "codehellchat")
	session.Values["username"] = username
	session.Values["email"] = email
	if err := session.Save(r, w); err != nil {
		log.Println(err)
		return
	}
}
