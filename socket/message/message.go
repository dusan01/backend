package message

type Message interface {
	Dispatch(Sender)
}

type Sender interface {
	Send([]byte)
}

type S map[string]interface{}
