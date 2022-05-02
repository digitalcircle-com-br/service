package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digitalcircle-com-br/buildinfo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type EMPTY_TYPE struct{}

func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second*10)
}

func IsDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

var svcName = ""
var sigCh = make(chan os.Signal)
var rediscli *redis.Client
var router *mux.Router
var onStop = func() {
	Log("Terminating")
}

func Init(s string) {
	svcName = s
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	redisurl := os.Getenv("REDIS")
	if redisurl == "" {
		redisurl = "redis://redis:6379"
	}
	opts, err := redis.ParseURL(redisurl)
	if err != nil {
		panic(err)
	}

	rediscli = redis.NewClient(opts)

	context, cancel := ctx()

	defer cancel()

	_, err = rediscli.Ping(context).Result()

	if err != nil {
		//TODO: improve error msg
		panic(err)
	}

	router = mux.NewRouter()

	router.Path("/__test").Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(buildinfo.String()))
	})

	router.Path("/__help").Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-type", "application/json")
		json.NewEncoder(w).Encode(apiEntries)
	})

	go func() {
		<-sigCh
		err := rediscli.Close()
		Err(err)
		onStop()
		if server != nil {
			HttpStop()
		}
	}()
	if IsDocker() {
		Log("Initiating Container for: %s", svcName)
	} else {
		Log("Initiating Service: %s", svcName)
	}
	Log(buildinfo.String())
}

var cfg = []byte{}

func Config(i interface{}) chan struct{} {
	ret := make(chan struct{})
	go func() {
		for {
			lastCfg := cfg
			cfgstr, err := DataGet(svcName)

			if err == nil {
				cfgbs := []byte(cfgstr)
				if !bytes.Equal(cfgbs, lastCfg) {
					cfg = cfgbs
					yaml.Unmarshal(cfg, i)
					ret <- struct{}{}
				}
			}

			time.Sleep(time.Duration(10) * time.Second)

		}
	}()
	return ret
}

func OnStop(f func()) {
	onStop = f
}

func LockMainRoutine() {
	for {
		time.Sleep(time.Minute)
	}
}

func ServerTiming(w http.ResponseWriter, metric string, desc string, t time.Time) {
	dur := time.Since(t)
	v := float64(dur.Nanoseconds()) / float64(1000000)
	w.Header().Add("Server-Timing", fmt.Sprintf("%s;desc=\"%s\";dur=%v", metric, desc, v))
	Debug("Server time: %s(%s): %v", desc, metric, v)
}
