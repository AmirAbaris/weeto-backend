package email

import "context"

type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Message struct {
	To      string
	Subject string
	HTML    string
}
