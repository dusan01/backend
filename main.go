package main

import (
  "flag"
  "fmt"
  "github.com/gorilla/pat"
  "hybris/debug"
  "hybris/routes"
  "log"
  "net/http"
  "runtime"
)

func init() {
  debugging := flag.Bool("debug", false, "Specifies whether or not logging is in debug mode")
  flag.Parse()
  debug.Debug = *debugging
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  router := pat.New()
  routes.Attach(router)

  fmt.Println("Loaded")

  log.Fatal(http.ListenAndServe(":38288", router))
}
