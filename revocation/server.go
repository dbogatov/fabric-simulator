package revocation

import (
	"fmt"
	"net/http"
)

func Hello() {
	fmt.Println("hello")
}

func runServer() {
	// logger.Notice("Server starting. Ctl+C to stop")

	http.HandleFunc("/", handleRevocationRequest)
	http.ListenAndServe(":8765", nil)
}

func handleRevocationRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}
