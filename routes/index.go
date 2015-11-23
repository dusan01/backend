package routes

import (
  "hybris/enums"
  "net/http"
)

func indexHandler(res http.ResponseWriter, req *http.Request) {
  res.Write([]byte(`turnf.fm backend, version ` + enums.Version))
}
