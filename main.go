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

	error_string := `. JSON string should be in this format: {"topic": [topic_name], "questions": [{question},{question},..]}
		(Disclaimer: do not use curly quotes, and the JSON string should be in the single line)`

	var questions models.QuestionsJson
	err := json.Unmarshal([]byte(fileData), &questions)
	if err != nil {
		return err.Error() + error_string
	}

	for _, q := range questions.Questions {
		if hasEmptyFields(q) {
			return `The generated questions have some empty fields, recheck them and generate again without empty fields.
			Each question must be a JSON object with these exact fields:{"QuestionId": [integer],"Ques": "[clear, specific question]",
			"OptionA": "[first option]","OptionB": "[second option]","OptionC": "[third option]","OptionD": "[fourth option]",
			"Answer": "[complete correct answer text matching one of the options exactly]"}` + error_string
		}
	}

	byte_questions, err := json.MarshalIndent(questions.Questions, "", " ")
	if err != nil {
		return err.Error() + error_string
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
	message_to_send.Content = `You are a Quiz Creator AI agent. You analyse the user input, research about it and then generate relevant high quality questions.

	Process:
	1. Analyse the user's input to identify the core topics related to it and then decide the one word queries.
	2. Gather enough information about the topics and related topics related to the queries, using tools
	**(first decide the one word query terms related to the user input and use web_search tool to gather basic the information about each query,
	then use llm_search to to get more info to deepen the reseach)**.
	3. Build a good and complete understanding of the topic and related topics with the help of the gathered information
	4. Generate the quiz questions accordingly in the specified format and save it in a json file.
	4. If you get any error in creating the json file, then analyse the error, think about why is it happening and what can you change in your json string to remove the error.

	Question Format:
	Each question must be a JSON string with these exact fields in single line:
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
	1. Tool: web_search  
	- Purpose: Returns a summary related to a general topic.  
	- Input format: A single word or a short keyword (e.g., "gravity", "Earth").  
	- Do NOT use full sentences or detailed questions.

	2. Tool: llm_search  
	- Purpose: Retrieves specific and detailed information about any topic.  
	- Input format: A complete query or descriptive sentence (e.g., "Explain quantum entanglement", "Why did the Mughal Empire decline?").

	3. Tool: json_file_creator  
	- Purpose: Creates a JSON file containing quiz questions.  
	- Input format: A single-line JSON string in this format:  
	{"topic": "Topic Name", "questions": [{question1}, {question2}, ...]}  
	- Rules:  
		- Use only straight quotes (") — no curly quotes (“ or ”)  
		- Keep the entire JSON on a single line (no line breaks or indentation)  
		- Ensure the JSON is valid and properly formatted
	Follow these formats exactly when calling the tools.

	You MUST think in this format:
	Thought: [your reasoning about what to do next]
	Action: tool_name
	Action Input: tool_Input_format

	**IMPORTANT: After Action Input, DO NOT GENERATE anything else.  
	DO NOT write "Observation:". The system will provide the observation.  
	You MUST STOP after Action Input.**

	After receiving the observation, continue:
	Thought: [reasoning about the observation]
	Action: [Next action if needed]

	You can continue this cycle as most 10 times or until you are satisfied that you have enough information to perform the task successfully

	Final Output:
	When task is complete, provide:
	Final Answer: [Task Successful/Unsuccessful - with brief explanation]

	Remember:
	-Always start with a Thought before taking any Action.
	-Try to use multiple tools before deciding to generate the questions.
	-Do Not try more that 10 times when stuck.
	-Always recheck and re-evaluate the questions for correct format and presence of all the fields.
	-Always try gathering enough information about the topic before generating the questions.`

	var user_input_message models.Message
	user_input_message.Role = "user"
	user_input_message.Content = "history, 10 questions"

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

			observation = tools.DuckDuckGoSearch(actionInput)

		} else if action == "json_file_creator" || strings.Contains(action, "json_file_creator") {
			fmt.Println("USING JSON FILE CREATOR TOOL")
			observation = createJSON(actionInput)
		} else if action == "llm_search" || strings.Contains(action, "llm_search") {
			fmt.Println("USING LLM SEARCH TOOL")

			var llmSearchMessage models.Message
			llmSearchMessage.Role = "assistant"
			llmSearchMessage.Content = "Give me most important and deep 30 points for this query without any follow ups : " + actionInput

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
// 	result := tools.DuckDuckGoSearch("The Great Akbar")

// 	fmt.Println(result)

// }
