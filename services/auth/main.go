package main

import (
	"fmt"
	"log"
	"net/http"
)

const version = "1"

func main() {
	err := InitDB()
	if err != nil {
		log.Fatal("Cant connect to DB:", err)
	}

	fmt.Println(
		"Auth Service запускается на порту 8081...",
	)

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/auth/register", registerHandler)
	http.HandleFunc("/api/auth/login", loginHandler)
	http.HandleFunc("/api/auth/me", authMiddleWare(meHandler))
	log.Fatal(http.ListenAndServe(":8081", nil))
}
