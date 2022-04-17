package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func CtxReq(c context.Context) *http.Request {
	raw := c.Value(CTX_REQ)
	return raw.(*http.Request)
}

func CtxRes(c context.Context) http.ResponseWriter {
	raw := c.Value(CTX_RES)
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
	ck, err := CtxReq(c).Cookie("X-SESSIONID")
	if err != nil {
		return ""
	}
	return ck.Value
}

func CtxDb(c context.Context) (db *gorm.DB, close func(), err error) {

	t := CtxTenant(c)
	if t == "" {
		return nil, nil, errors.New("tenant not found")
	}
	db, err = DBN(t)
	return
}

func CtxVars(c context.Context) map[string]string {
	return mux.Vars(CtxReq(c))
}

func CtxDone(c context.Context) func() {
	raw := c.Value(CTX_DONE)
	return raw.(func())
}

func CtxErr(c context.Context, err error) bool {
	if err != nil {
		Err(err)
		http.Error(CtxRes(c), err.Error(), http.StatusInternalServerError)
		return true
	}
	return false
}

func CtxSession(c context.Context) *Session {
	return c.Value(CTX_SESSION).(*Session)
}
