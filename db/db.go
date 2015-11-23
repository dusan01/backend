package db

import (
	"time"

	uppdb "upper.io/db"
	"upper.io/db/mongo"
)

var Session uppdb.Database

const (
	CacheExpiration      time.Duration = 0
	CacheCleanupInterval time.Duration = 60 * time.Minute
)

func init() {
	sess, err := uppdb.Open(mongo.Adapter, mongo.ConnectionURL{
		Address:  uppdb.Host("127.0.0.1"),
		Database: "hybris",
	})

	if err != nil {
		panic(err)
	}

	Session = sess
}
