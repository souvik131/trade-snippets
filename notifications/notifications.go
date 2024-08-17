package notifications

type Messenger interface {
	Send(string)
}
