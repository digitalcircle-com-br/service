package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

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

func HttpHandle[TIN, TOUT any](hpath string, method string, perm string, f func(context.Context, TIN) (TOUT, error)) {
	router.Path(hpath).Methods(method).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if perm != "" {
		// 	sess := r.Header.Get("X-SESSION")
		// 	_, err := DataHGet(sess, perm)
		// 	if err != nil {
		// 		_, err := DataHGet(sess, "*")
		// 		if err != nil {
		// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		// 			return
		// 		}
		// 	}
		// }
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
