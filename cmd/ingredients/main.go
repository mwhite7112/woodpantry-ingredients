package main

import (
	"net/http"

	"github.com/mwhite7112/woodpantry-ingredients/internal/api"
)

func main() {
	handler := api.NewRouter()
	http.ListenAndServe(":8000", handler)
}
