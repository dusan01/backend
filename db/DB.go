package db

import (
  "gopkg.in/mgo.v2"
)

var DB *mgo.Database

func init() {
  session, err := mgo.Dial("127.0.0.1")
  if err != nil {
    panic(err)
  }

  DB = session.DB("hybris")
}
