package models

type Question struct {
	QuestionId int    `json:"QuestionId"`
	Ques       string `json:"Ques"`
	OptionA    string `json:"OptionA"`
	OptionB    string `json:"OptionB"`
	OptionC    string `json:"OptionC"`
	OptionD    string `json:"OptionD"`
	Answer     string `json:"Answer"`
}

type QuestionsJson struct {
	Topic     string
	Questions []Question
}
