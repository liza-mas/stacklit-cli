package main

import (
	"fmt"
	"github.com/example/pkg/handler"
	"net/http"
)

func main() {
	fmt.Println("hello")
	http.ListenAndServe(":8080", nil)
}
