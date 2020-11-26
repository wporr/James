package main

import (
    "fmt"
    "log"
    "net/http"
    "encoding/json"
)

func main() {
    routes()
    fmt.Println("Starting server...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func routes() {
    http.HandleFunc("/webhook/twitter", func(w http.ResponseWriter, r *http.Request) {
        crcToken, ok := r.URL.Query()["crc_token"]
        if !ok {
            fmt.Println("Couldnt get crc_token")
        }

        responseToken := generateResponseToken([]byte(crcToken[0]))
        resp, err := json.Marshal(map[string]string{
            "response_token": responseToken,
        })
        check(err)

        fmt.Fprintln(w, resp)
    })
}

// For checking errors more easily
func check(e error) {
	if e != nil {
		panic(e)
	}
}
