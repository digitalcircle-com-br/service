package service

import (
	"fmt"
	"log"
)

func Log(s string, p ...interface{}) {

	log.Printf(fmt.Sprintf("[%s] - %s", svcName, s), p...)
}

func Debug(s string, p ...interface{}) {
	log.Printf(fmt.Sprintf("DBG [%s] - %s", svcName, s), p...)
}

func Fatal(s ...interface{}) {
	log.Fatal(s...)
}

func Err(err error) {
	if err != nil {
		Log("Error: %s", err.Error())
	}
}
