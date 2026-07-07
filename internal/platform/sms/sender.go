package sms

import "context"

type Parameter struct {
	Name  string
	Value string
}

type Sender interface {
	VerifySend(ctx context.Context, mobile string, templateID int, params []Parameter) (messageID int64, cost float64, err error)
}
