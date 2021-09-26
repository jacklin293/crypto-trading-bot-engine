package message

import (
	"errors"
	"fmt"
)

type Messenger interface {
	Send(int64, string)
}

func NewSender(name string, data map[string]interface{}) (m Messenger, err error) {
	switch name {
	case "telegram":
		token, ok := data["token"].(string)
		if !ok {
			err = errors.New("'token' is missing")
			return
		}
		return newTelegram(token)
	}
	err = fmt.Errorf("sender '%s' no supported", name)
	return
}
