package slack

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var SlackToken string

func InitSlackClient() {
	SlackToken = os.Getenv("SLACK_TOKEN")
	if SlackToken == "" {
		log.Fatal("Missing SLACK_TOKEN in environment variables")
	}
}

func Fetch(endpoint string, params map[string]string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", os.Getenv("SLACK_API_URI")+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+SlackToken)

	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	return ioutil.ReadAll(resp.Body)
}
