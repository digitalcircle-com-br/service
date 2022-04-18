package service_test

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/digitalcircle-com-br/service"
)

func TestPub(t *testing.T) {
	service.Init("test")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		chMsg, close := service.Sub("ach")
		for i := 0; i < 3; i++ {
			m := <- chMsg
			log.Printf("%#v", m)
		}
		close()
		log.Printf("Done Listening")
		wg.Done()

	}()
	go func() {
		time.Sleep(time.Millisecond * 10)
		for i := 0; i < 3; i++ {
			service.Pub("ach", fmt.Sprintf("Msg: %v - %s", i, time.Now().String()))
		}
		log.Printf("Done Sending")
		wg.Done()
	}()
	wg.Wait()
	log.Printf("Finished")
}
