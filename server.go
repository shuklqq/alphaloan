package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/alphaloan/vehicle/internal/database"
)

func main() {
    db, err := database.Init("alphaloan.db")
    if err != nil {
        log.Fatalf("init database: %v", err)
    }
    defer db.Close()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)
    })

    http.ListenAndServe(":3000", nil)
}