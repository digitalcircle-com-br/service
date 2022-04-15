package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digitalcircle-com-br/buildinfo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	rediscli = redis.NewClient(&redis.Options{Addr: "redis:6379", Password: os.Getenv("KEY")})

	context, cancel := ctx()
	defer cancel()
	_, err := rediscli.Ping(context).Result()

	if err != nil {
		panic(err)
	}

	router = mux.NewRouter()

	router.Path("/__test").Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(buildinfo.String()))
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

func DataSet(k string, v interface{}, to int) error {
	context, cancel := ctx()
	defer cancel()
	return rediscli.Set(context, k, v, time.Duration(to)*time.Second).Err()
}

func DataGet(k string) (string, error) {
	context, cancel := ctx()
	defer cancel()
	return rediscli.Get(context, k).Result()
}

func DataDel(k string) (int64, error) {
	context, cancel := ctx()
	defer cancel()
	return rediscli.Del(context, k).Result()
}

func DataHSet(k string, v ...interface{}) (int64, error) {
	context, cancel := ctx()
	defer cancel()
	return rediscli.HSet(context, k, v...).Result()
}

func DataHGet(k string, v string) (string, error) {
	context, cancel := ctx()
	defer cancel()
	return rediscli.HGet(context, k, v).Result()
}

func DataHDel(k string, v ...string) (int64, error) {
	context, cancel := ctx()
	defer cancel()
	return rediscli.HDel(context, k, v...).Result()
}

func DataHGetAll(k string) (map[string]string, error) {
	context, cancel := ctx()
	defer cancel()
	return rediscli.HGetAll(context, k).Result()
}

func Enqueue(q string, v ...interface{}) error {
	context, cancel := ctx()
	defer cancel()
	return rediscli.LPush(context, q, v...).Err()
}

func Dequeue(q string, to int) ([]string, error) {
	context := context.Background()
	return rediscli.BRPop(context, time.Second*time.Duration(to), q).Result()
}

type Msg struct {
	Chan    string
	Payload string
	Err     error
}

func Sub(ch ...string) (ret <-chan *Msg, closefn func()) {
	sub := rediscli.Subscribe(context.Background(), ch...)
	inret := make(chan *Msg)
	ret = inret
	run := true
	go func() {
		for run {
			msg, err := sub.ReceiveMessage(context.Background())
			m := &Msg{Chan: msg.Channel, Payload: msg.Payload, Err: err}
			inret <- m
		}
	}()

	closefn = func() {
		run = false
		sub.Close()
		close(inret)
	}
	return
}

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

func OnStop(f func()) {
	onStop = f
}

func LockMainRoutine() {
	for {
		time.Sleep(time.Minute)
	}
}

func DB() (ret *gorm.DB, err error) {
	return DBN("default")
}

func DBN(n string) (ret *gorm.DB, err error) {
	dsn, err := DataHGet("dsn", n)

	if err != nil {
		return
	}

	ret, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{})

	if err != nil {
		return
	}

	err = ret.Raw("select 1+1").Error

	return
}

func CtxReq(c context.Context) *http.Request {
	raw := c.Value("REQ")
	return raw.(*http.Request)
}

func CtxRes(c context.Context) http.ResponseWriter {
	raw := c.Value("RES")
	return raw.(http.ResponseWriter)
}

func CtxSessionID(c context.Context) string {
	ck, err := CtxReq(c).Cookie("X-SESSIONID")
	if err != nil {
		return ""
	}
	return ck.Value
}

func CtxTenant(c context.Context) string {
	sess := CtxSessionID(c)
	t, err := DataHGet(sess, "tenant")
	if err != nil {
		return ""
	}
	return t
}

func CtxDb(c context.Context) (db *gorm.DB, err error) {

	t := CtxTenant(c)
	if t == "" {
		return nil, errors.New("tenant not found")
	}
	db, err = DBN(t)
	return
}

func CtxVars(c context.Context) map[string]string {
	return mux.Vars(CtxReq(c))
}

func CtxDone(c context.Context) func() {
	raw := c.Value("DONE")
	return raw.(func())
}

func HttpHandle[TIN, TOUT any](hpath string, method string, perm string, f func(context.Context, TIN) (TOUT, error)) {
	router.Path(hpath).Methods(method).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if perm != "" {
			sess := r.Header.Get("X-SESSION")
			_, err := DataHGet(sess, perm)
			if err != nil {
				_, err := DataHGet(sess, "*")
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
		}
		done := false
		nctx := context.WithValue(r.Context(), "REQ", r)
		nctx = context.WithValue(nctx, "RES", w)
		nctx = context.WithValue(nctx, "DONE", func() {
			done = true
		})
		nctx.Done()

		r = r.WithContext(nctx)
		in := new(TIN)

		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out, err := f(r.Context(), *in)
		if !done {
			w.Header().Add("Content-Type", "application/json")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(out)
		}

	})
}

var server *http.Server

func HttpRun(addr string) {
	HttpStart(addr)
	LockMainRoutine()
}

func HttpStart(addr string) *http.Server {
	if addr == "" {
		addr = ":8080"
	}
	Log("Server will listen at: %s", addr)
	server = &http.Server{Addr: addr, Handler: router}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			Err(err)
		}
		server = nil
	}()
	return server
}

func HttpStop() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	server.Shutdown(ctx)
	server = nil
}

func HttpRouter() *mux.Router {
	return router
}

func ServerTiming(w http.ResponseWriter, metric string, desc string, t time.Time) {
	dur := time.Since(t)
	v := float64(dur.Nanoseconds()) / float64(1000000)
	w.Header().Add("Server-Timing", fmt.Sprintf("%s;desc=\"%s\";dur=%v", metric, desc, v))
	Debug("Server time: %s(%s): %v", desc, metric, v)
}

func CtxErr(c context.Context, err error) bool {
	if err != nil {
		Err(err)
		http.Error(CtxRes(c), err.Error(), http.StatusInternalServerError)
		return true
	}
	return false
}
