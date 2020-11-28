package main

import (
    "fmt"
    "log"
    "syscall"
    "os"
    "os/signal"
    "net/http"
)

func main() {
	fmt.Println("James v0.01")
    routes()
    fmt.Println("Starting server...")

	// Begin handling messages from the stream
	go func(){log.Fatal(http.ListenAndServe(":8080", nil))}()

    fmt.Println("Registering Webhook")
    registerWebhook()
	// Wait for SIGING and SIGTERM (ctrl-c)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	fmt.Println("Stopping server...")
}

func routes() {
    http.HandleFunc("/webhook/twitter", webhookHandler)
}

// For checking errors more easily
func check(e error) {
	if e != nil {
		panic(e)
	}
}
