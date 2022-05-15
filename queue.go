package service

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type qmsg struct {
	Ret     string
	Payload interface{}
}

func qserveOnce(q string, f func(i interface{}) (o interface{}, err error)) error {
	context, cancel := ctx()
	defer cancel()

	cmd := rediscli.BRPop(context, time.Second*0, q)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	strs, err := cmd.Result()

	if err != nil {
		return err
	}

	msg := &qmsg{}
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(strs[1]), msg)

	if err != nil {
		return err
	}

	o, err := f(msg.Payload)

	if err != nil {
		return err
	}

	bs, err := json.Marshal(o)
	if err != nil {
		return err
	}
	err = rediscli.LPush(context, msg.Ret, bs).Err()

	return err
}

func QServe(q string, f func(i interface{}) (o interface{}, err error)) {
	for {
		err := qserveOnce(q, f)
		if err != nil {
			Err(err)
			time.Sleep(time.Second)
		}
	}
}

func QSend(q string, i interface{}) error {
	context, cancel := ctx()
	msg := qmsg{uuid.NewString(), i}
	defer cancel()
	cmd := rediscli.LPush(context, q, msg)
	return cmd.Err()
}

func QRpc(q string, i interface{}, o interface{}) error {
	context, cancel := ctx()
	defer cancel()
	msg := qmsg{uuid.NewString(), i}

	bsmsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = rediscli.LPush(context, q, bsmsg).Err()
	if err != nil {
		return err
	}

	cmdret := rediscli.BRPop(context, time.Second*30, msg.Ret)
	if cmdret.Err() != nil {
		return cmdret.Err()
	}
	strs, err := cmdret.Result()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(strs[1]), o)
	return err
}
