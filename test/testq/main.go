package main

import (
	"log"

	"github.com/digitalcircle-com-br/service"
)

func main() {
	service.Init("testq")
	go service.QServe("some", func(i interface{}) (o interface{}, err error) {
		log.Printf("%#v", i)
		return "OK", nil
	})

	var s string
	err := service.QRpc("some", "asd", &s)
	if err != nil {
		service.Err(err)
	} else {
		log.Printf(s)
	}
}
