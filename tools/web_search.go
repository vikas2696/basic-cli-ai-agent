package tools

import (
	"Go-ReAct-basic-AI-agent-project/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func fetchMainTextFromURL(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}

	// Extract only paragraphs from the article
	var content string
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		paragraph := s.Text()
		if len(paragraph) > 50 { // Filter out very short or empty lines
			content += paragraph + "\n\n"
		}
	})

	return content, nil
}

func naiveSummarize(text string, maxSentences int) string {
	sentences := strings.Split(text, ".")
	summary := ""
	count := 0

	for _, s := range sentences {
		if strings.Contains(s, "important") || strings.Contains(s, "main") || len(s) > 100 {
			summary += strings.TrimSpace(s) + ". "
			count++
			if count >= maxSentences {
				break
			}
		}
	}

	return summary
}

func DuckDuckGoSearch(query string) string {
	search_query := url.QueryEscape(query)
	DDGapiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_redirect=1&no_html=1", search_query)

	resp, err := http.Get(DDGapiURL)
	if err != nil {
		return err.Error() + ".Try giving more specific query"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result models.DDGSearchResult

	err = json.Unmarshal(body, &result)
	if err != nil {
		return err.Error() + ".Try again"
	}

	//return summary, nil

	if result.AbstractURL != "" {
		main_text, err := fetchMainTextFromURL(result.AbstractURL)
		if err != nil {
			return err.Error() + ".Try again" + result.AbstractURL
		}

		summary := naiveSummarize(main_text, 30)
		return (summary)
	}

	return "sorry, No result found, try using **one word query** or try using another tool, if you have a specific muti-word query."
}
