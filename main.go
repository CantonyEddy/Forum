package main

import (
	"Forum/BDD"
	"Forum/handlers"
)

func main() {
	BDD.InitTable()
	handlers.StartServer()
}
