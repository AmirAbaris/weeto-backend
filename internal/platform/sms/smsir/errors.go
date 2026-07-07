package smsir

import "errors"

var (
	ErrAPIFailure = errors.New("sms.ir api request failed")
	ErrRejected   = errors.New("sms.ir rejected request")
)
