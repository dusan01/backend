package main

import (
  "flag"
  "github.com/gorilla/pat"
  "hybris/routes"
  "log"
  "net/http"
  "runtime"
)

func main() {
  flag.Parse()
  runtime.GOMAXPROCS(runtime.NumCPU())

  router := pat.New()
  routes.Attach(router)

  log.Fatal(http.ListenAndServe(":38288", router))
}
