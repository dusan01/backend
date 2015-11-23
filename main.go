package main

import (
	"flag"
	"hybris/routes"
	"log"
	"net/http"
	"runtime"

	"github.com/gorilla/pat"

	_ "hybris/debug"
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	router := pat.New()
	routes.Attach(router)

	log.Fatal(http.ListenAndServe(":38288", router))
}
