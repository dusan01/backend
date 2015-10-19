package main

import (
  "fmt"
  "github.com/gorilla/mux"
  "hybris/routes"
  "net/http"
  "runtime"
)

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  router := mux.NewRouter()
  routes.Attach(router)

  fmt.Println("Loaded")

  fmt.Println(http.ListenAndServe(":9002", router))
}
