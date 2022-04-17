package service

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func sessionKey(t string, id string) string {
	return fmt.Sprintf("session:%s:%s", t, id)
}

func sessionKeyFromId(rawid string) (t string, sid string, hash []byte, err error) {
	rawdec, err := base64.StdEncoding.DecodeString(rawid)
	if err != nil {
		return
	}

	parts := strings.Split(string(rawdec), ".")
	if len(parts) != 3 {
		err = errors.New("session id in wrong format")
		return
	}
	t = parts[0]
	sid = parts[1]

	hash, err = base64.StdEncoding.DecodeString(parts[2])
	return
}

func SessionSave(s *Session) (id string, err error) {
	sid := uuid.NewString()
	s.Sessionid = sid
	sessbs, _ := json.Marshal(s)
	hash := md5.New()
	hash.Write(sessbs)
	sum := hash.Sum(nil)
	hashEnc := base64.StdEncoding.EncodeToString(sum)
	iddec := fmt.Sprintf("%s.%s.%s", s.Tenant, s.Sessionid, hashEnc)
	id = base64.StdEncoding.EncodeToString([]byte(iddec))

	k := sessionKey(s.Tenant, s.Sessionid)
	err = DataSet(k, sessbs, 0)
	return
}

func SessionLoad(rawid string) (sess *Session, err error) {

	t, id, hash, err := sessionKeyFromId(rawid)
	if err != nil {
		return
	}

	k := sessionKey(t, id)
	str, err := DataGet(k)

	if err != nil {
		return nil, err
	}
	hasher := md5.New()
	hasher.Write([]byte(str))
	hashVal := hasher.Sum(nil)
	//"\xbd\xf03\xc6\xe7\xe0\x0eO\x10<T\x10V\r\x0f\xa5"
	//"\xbd\xf03\xc6\xe7\xe0\x0eO\x10<T\x10V\r\x0f\xa5"
	if !bytes.Equal(hash, hashVal) {
		err = errors.New("session hash invalid")
		return
	}

	ret := &Session{}
	err = json.Unmarshal([]byte(str), ret)
	return ret, err
}

func SessionDel(rawid string) (ret int64, err error) {
	t, id, _, err := sessionKeyFromId(rawid)
	if err != nil {
		return
	}
	k := sessionKey(t, id)
	return DataDel(k)
}

func SessionEnc(s *Session) (id string, sessbs []byte) {
	sessbs, _ = json.Marshal(s)
	hasher := md5.New()
	hasher.Write(sessbs)
	sum := hasher.Sum(nil)
	hashEnc := base64.StdEncoding.EncodeToString(sum)
	iddec := fmt.Sprintf("%s.%s.%s", s.Tenant, s.Sessionid, hashEnc)
	id = base64.StdEncoding.EncodeToString([]byte(iddec))
	return
}

func SessionDec(s string) (t string, id string, hash []byte, err error) {
	rawdec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return
	}
	parts := strings.Split(string(rawdec), ".")
	if len(parts) != 3 {
		err = errors.New("session id in wrong format")
		return
	}
	t = parts[0]
	id = parts[1]

	hash, err = base64.StdEncoding.DecodeString(parts[2])
	return

}
