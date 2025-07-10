package main

import (
	"Go-ReAct-basic-AI-agent-project/models"
	"Go-ReAct-basic-AI-agent-project/tools"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func parseResponse(content string) (string, string, string, string, error) {
	lines := strings.Split(content, "\n")

	var thought, action, actionInput, finalResult string

	for _, line := range lines {
		if strings.HasPrefix(line, "Thought: ") {
			thought = strings.TrimPrefix(line, "Thought: ")
		} else if strings.HasPrefix(line, "Action: ") {
			action = strings.TrimPrefix(line, "Action: ")
		} else if strings.HasPrefix(line, "Action Input: ") {
			actionInput = strings.TrimPrefix(line, "Action Input: ")
		} else if strings.HasPrefix(line, "Final Result: ") {
			finalResult = strings.TrimPrefix(line, "Final Result: ")
		}
	}

	return thought, action, actionInput, finalResult, nil
}

func read_context_file(filename string) ([]models.Message, error) {

	var previous_messages []models.Message

	previous_messages_bytes, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("File reading error")
		return previous_messages, err
	}

	err = json.Unmarshal(previous_messages_bytes, &previous_messages)
	if err != nil {
		fmt.Println("Unmarshalling error")
		return previous_messages, err
	}

	return previous_messages, err
}

func write_context_file(filename string, messages []models.Message) error {

	byte_messages, err := json.MarshalIndent(messages, "", " ")
	if err != nil {
		fmt.Println("Error marshaling:", err)
		return err
	}

	err = os.WriteFile(filename, byte_messages, 0644)
	if err != nil {
		fmt.Println("File writing error")
		return err
	}
	return nil
}

func LLMcall(messages []models.Message) (map[string]any, error) {

	llm_endpoint_url := "http://localhost:11434/api/chat"
	var result map[string]any

	llm_request_body := models.RequestBody{
		Model:    "gemma3",
		Messages: messages,
		Stream:   false,
	}

	json_request_body, err := json.Marshal(llm_request_body)
	if err != nil {
		fmt.Println("Error marshaling:", err)
		return result, err
	}

	response, err := http.Post(llm_endpoint_url, "application/json", bytes.NewBuffer(json_request_body))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return result, err
	}

	json.NewDecoder(response.Body).Decode(&result)
	return result, err
}

func convertLLMResult(result map[string]any) models.Message {
	var received_message models.Message
	full_message := result["message"].(map[string]any)
	received_message.Role = full_message["role"].(string)
	received_message.Content = full_message["content"].(string)
	return received_message
}

func hasEmptyFields(q models.Question) bool {
	return q.Ques == "" || q.OptionA == "" || q.OptionB == "" || q.OptionC == "" || q.OptionD == "" || q.Answer == ""
}

func createJSON(fileData string) string {

	var questions models.QuestionsJson
	err := json.Unmarshal([]byte(fileData), &questions)
	if err != nil {
		return err.Error() + " try again"
	}

	for _, q := range questions.Questions {
		if hasEmptyFields(q) {
			return "The generated questions have some empty fields, recheck them and generate again without empty fields"
		}
	}

	byte_questions, err := json.MarshalIndent(questions.Questions, "", " ")
	if err != nil {
		return err.Error() + " try again"
	}

	err = os.WriteFile(questions.Topic+".json", byte_questions, 0644)
	if err != nil {
		return err.Error()
	}
	return "json file created successfully, now you can give the Final Answer"

}

func main() {

	var message_to_send models.Message

	previous_messages, err := read_context_file("conversation.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	all_messages := previous_messages

	message_to_send.Role = "system"
	message_to_send.Content =
		`You are a Quiz Creator AI agent. You analyse the user input, research about it and then generate relevant high quality questions.

	Process:
	1. Analyse the user's input to identify the core topics related to it.
	2. Gather enough information about the topic and related topics using tools.
	3. Build a good and complete understanding of the topic and related topics with the help of the gathered information
	4. Generate the quiz questions accordingly in the specified format and save it in a json file.
	4. If you get any error in creating the json file, then analyse the error, think about why is it happening and what can you change in your json string to remove the error.

	Question Format:
	Each question must be a JSON object with these exact fields:
	{
		"QuestionId": [integer],
		"Ques": "[clear, specific question]",
		"OptionA": "[first option]",
		"OptionB": "[second option]",
		"OptionC": "[third option]",
		"OptionD": "[fourth option]",
		"Answer": "[complete correct answer text matching one of the options exactly]"
	}
	for example, 
	{
		"QuestionId": 1,
		"Ques": "What is a qubit?",
		"OptionA": "A bit that can be either 0 or 1.",
		"OptionB": "A bit that can exist in multiple states simultaneously.",
		"OptionC": "A physical wire used to transmit quantum data.",
		"OptionD": "A type of classical computer.",
		"Answer": "A bit that can exist in multiple states simultaneously."
	}

	Available Tools:
	- web_search: lets you search a query on the search engine for your research purpose. Input format: your search query
	- llm_search: gives you basic information about the topic. Input format: your search query
	- json_file_creator: creates a json file of the questions you will provide, takes input in the form of a JSON string of single line. 
						 Input format: {"topic": [topic_name], "questions":[{question},{question},..]}(Disclaimer: do not use curly quotes)

	Tool Selection Rules:

	Use web_search when:
	- You need current, real-time information
	- Looking for recent developments or news
	- Researching factual data, statistics, or specific details
	- Finding multiple perspectives on a topic
	- Gathering comprehensive information about a subject

	Use llm_search when:
	- if you are not satisfied with the web_search tool or need more information
	- You need to refine or clarify information already gathered
	- Looking for specific details within a large dataset
	- Need to cross-reference or verify information
	- Searching for patterns or connections in collected data
	
	You MUST think in this format:
	Thought: [your reasoning about what to do next]
	Action: tool_name
	Action Input: tool_Input_format

	STOP HERE Do not generate "Observation:" - the system will provide it.

	After receiving the observation, continue:
	Thought: [reasoning about the observation]
	Action: [Next action if needed]

	Final Output:
	When task is complete, provide:
	Final Answer: [Task Successful/Unsuccessful - with brief explanation]

	Remember:
	-Always start with a Thought before taking any Action.
	-Try to use multiple tools before deciding to generate the questions.
	-Always recheck and re-evaluate the questions for correct format and presence of all the fields.
	-Always try gathering enough information about the topic before generating the questions.`

	var user_input_message models.Message
	user_input_message.Role = "user"
	user_input_message.Content = "quiz about simple science, 5 questions, very easy"

	all_messages = append(all_messages, message_to_send)
	all_messages = append(all_messages, user_input_message)

	result, err := LLMcall(all_messages) //calling LLM
	if err != nil {
		fmt.Println("Error calling LLM", err)
		return
	}

	received_message := convertLLMResult(result) // converting LLM result
	fmt.Println(received_message.Content)

	all_messages = append(all_messages, received_message) // adding received_message to the conversation file for next time usage
	err = write_context_file("conversation.json", all_messages)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	_, action, actionInput, _, err := parseResponse(received_message.Content)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	found := strings.Contains(received_message.Content, "Final Answer")
	observation := "no observations yet"

	for !found {
		fmt.Println("TRYING TO USE THE TOOLS")
		if action == "web_search" || strings.Contains(action, "web_search") {
			fmt.Println("USING WEB SEARCH TOOL")

			observation, err = tools.DuckDuckGoSearch(actionInput)
			if err != nil {
				fmt.Println(err.Error())
				return
			}

		} else if action == "json_file_creator" || strings.Contains(action, "json_file_creator") {
			fmt.Println("USING JSON FILE CREATOR TOOL")
			observation = createJSON(actionInput)
		} else if action == "llm_search" || strings.Contains(action, "llm_search") {
			fmt.Println("USING LLM SEARCH TOOL")

			var llmSearchMessage models.Message
			llmSearchMessage.Role = "assistant"
			llmSearchMessage.Content = "give me one concise paragraph and quality information without any follow-up, which will help me with : " + actionInput

			llmSearchMessages := []models.Message{llmSearchMessage}

			result, err := LLMcall(llmSearchMessages) //calling LLM
			if err != nil {
				fmt.Println("Error calling LLM", err)
				return
			}

			llm__search_response := convertLLMResult(result)
			observation = llm__search_response.Content
		}

		var new_message models.Message
		new_message.Role = "user"
		new_message.Content = "Observation: " + observation
		all_messages = append(all_messages, new_message)
		fmt.Println(new_message.Content)

		result, err := LLMcall(all_messages) //calling LLM
		if err != nil {
			fmt.Println("Error calling LLM", err)
			return
		}

		received_message := convertLLMResult(result) // converting LLM result
		fmt.Println(received_message.Content)

		found = strings.Contains(received_message.Content, "Final Answer")

		all_messages = append(all_messages, received_message) // adding received_message to the conversation file for context

		_, action, actionInput, _, err = parseResponse(received_message.Content)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

	}

	err = write_context_file("conversation.json", all_messages)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

// func main() {
// 	result, err := tools.DuckDuckGoSearch("Quantum Computing")
// 	if err != nil {
// 		fmt.Println(err.Error())
// 	} else {
// 		fmt.Println(result)
// 	}
// }
