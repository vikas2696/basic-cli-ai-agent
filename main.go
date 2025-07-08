package main

import (
	"Go-ReAct-basic-AI-agent-project/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {

	llm_endpoint_url := "http://localhost:11434/api/chat"

	var message models.Message

	message.Role = "user"
	message.Content = `You are a quiz question generator ai agent who is going to find trending and relevant topics form the web and 
						then research about that topic and then generatae quiz questions about that topic`

	messages := []models.Message{message}

	llm_request_body := models.RequestBody{
		Model:    "gemma3",
		Messages: messages,
		Stream:   false,
	}

	json_request_body, err := json.Marshal(llm_request_body)
	if err != nil {
		fmt.Println("Error marshaling:", err)
		return
	}

	response, err := http.Post(llm_endpoint_url, "application/json", bytes.NewBuffer(json_request_body))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return
	}

	var result map[string]any
	json.NewDecoder(response.Body).Decode(&result)

	fmt.Print(result)
	var received_message models.Message
	full_messages := result["message"].(map[string]any)
	received_message.Role = full_messages["role"].(string)
	received_message.Content = full_messages["content"].(string)

	messages = append(messages, message)
	messages = append(messages, received_message)

	byte_messages, err := json.Marshal(messages)
	if err != nil {
		fmt.Println("Error marshaling:", err)
		return
	}

	os.WriteFile("conversation.json", byte_messages, 0644)
}
