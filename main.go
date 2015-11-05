package main

import (
  "flag"
  "github.com/gorilla/pat"
  "gopkg.in/mgo.v2"
  "hybris/db"
  "hybris/debug"
  "hybris/pool"
  "hybris/routes"
  "log"
  "net/http"
  "runtime"
)

func main() {
  flag.Parse()
  runtime.GOMAXPROCS(runtime.NumCPU())

  debug.Log("Creating and attaching routes")
  router := pat.New()
  routes.Attach(router)

  debug.Log("Loading communities into memory")
  session, err := mgo.Dial("127.0.0.1")
  if err != nil {
    panic(err)
  }

  DB := session.DB("hybris")

  iter := DB.C("communities").Find(nil).Iter()
  var result db.Community
  for iter.Next(&result) {
    r := result
    _ = pool.NewCommunity(&r)
    debug.Log("Loaded community %s", r.Id)
  }

  debug.Log("Finished")

  log.Fatal(http.ListenAndServe(":38288", router))
}
