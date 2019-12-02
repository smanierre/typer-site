package main

import (
	"log"
	"net/http"

	"github.com/smanierre/typer-site/server"
	"github.com/smanierre/typer-site/store"
	db "github.com/smanierre/typer-site/store/postgres"
)

func main() {
	store, err := store.NewStore()
	db.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	s := server.NewServer(store)
	s.Handle("/", http.FileServer(http.Dir("public")))
	log.Fatal(http.ListenAndServe(":8080", s))
}
