package models

type RequestBody struct {
	Model    string
	Messages []Message
	Stream   bool
}

type Message struct {
	Role    string
	Content string
}
