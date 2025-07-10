package tools

import (
	"Go-ReAct-basic-AI-agent-project/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func DuckDuckGoSearch(query string) (string, error) {
	search_query := url.QueryEscape(query)
	DDGapiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_redirect=1&no_html=1", search_query)

	resp, err := http.Get(DDGapiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result models.DDGSearchResult

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	if result.Abstract != "" {
		return fmt.Sprintf("%s\n%s", result.Heading, result.Abstract), nil
	}

	return "No direct answer found.", nil
}
