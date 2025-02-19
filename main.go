package main

import (
	"fmt"
	"log"
	"net/http"
	"todo-api/database"
	"todo-api/routes"
)

func main() {
	database.ConnectDB()
	router := routes.SetupRoutes()
	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
