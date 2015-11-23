package routes

import (
	"fmt"
	"hybris/socket"
	"net/http"
)

func socketHandler(res http.ResponseWriter, req *http.Request) {
	_, err := socket.New(res, req)
	if err != nil {
		fmt.Println(err.Error())
	}
}
