package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

var tmpl *template.Template

func StartServer() {
	var err error

	tmpl, err = template.New("newgame").ParseFiles("web/HTML/NewGame.html")
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

	fmt.Println("Pour accéder à la page web -> http://localhost:8080/")
	err1 := http.ListenAndServe(":8080", nil)
	if err1 != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
