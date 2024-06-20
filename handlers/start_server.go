package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

var tmpl *template.Template
var tmpl_register *template.Template
var tmpl_login *template.Template

func StartServer() {
	var err error

	tmpl, err = template.New("index").ParseFiles("Templates/index.html")
	if err != nil {
		panic(err)
	}

	tmpl_register, err = template.New("register").ParseFiles("Templates/register.html")
	if err != nil {
		panic(err)
	}

	tmpl_login, err = template.New("login").ParseFiles("Templates/login.html")
	if err != nil {
		panic(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.Dir(wd + "\\web"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, wd+"\\Templates\\index.html")
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/register" {
			http.ServeFile(w, r, wd+"\\Templates\\register.html")
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.ServeFile(w, r, wd+"\\Templates\\login.html")
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	fmt.Println("Pour accéder à la page web -> http://localhost:8080/")
	err1 := http.ListenAndServe(":8080", nil)
	if err1 != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
