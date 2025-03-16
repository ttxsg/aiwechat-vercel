package api

import (
    "fmt"
    "net/http"
)

// Handler 函数是 Vercel 的入口点
func Handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}
