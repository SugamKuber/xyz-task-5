package slack

import (
	"encoding/json"
	"fmt"

	"os"

	"xyz-task-5/internal/models"
)

type Message struct {
	Text      string `json:"text"`
	User      string `json:"user"`
	Timestamp string `json:"ts"`
	ThreadTS  string `json:"thread_ts,omitempty"`
}

type ChannelResponse struct {
	Ok       bool      `json:"ok"`
	Messages []Message `json:"messages"`
	Channels []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channels"`
	Members []string `json:"members"`
	Error   string   `json:"error"`
}

func FetchChannelMessages(channelID, channelName string) ([]models.LogMessage, error) {
	params := map[string]string{
		"channel": channelID,
		"limit":   "1000",
	}

	data, err := Fetch("conversations.history", params)
	if err != nil {
		return nil, fmt.Errorf("error fetching messages for channel %s: %v", channelID, err)
	}

	var result ChannelResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse messages response for channel %s: %v", channelID, err)
	}

	if !result.Ok {
		return nil, fmt.Errorf("Slack API error fetching messages for channel %s: %s", channelID, result.Error)
	}

	var messages []models.LogMessage
	for _, msg := range result.Messages {
		logMsg := models.LogMessage{
			Text:      msg.Text,
			User:      msg.User,
			Timestamp: msg.Timestamp,
		}
		messages = append(messages, logMsg)
	}

	return messages, nil
}

func FetchAllReplies(channelID, channelName, mainMessageTS string) []models.LogMessage {
	params := map[string]string{
		"channel": channelID,
		"ts":      mainMessageTS,
	}

	data, err := Fetch("conversations.replies", params)
	if err != nil {
		return nil
	}

	var result ChannelResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	var replies []models.LogMessage

	for _, msg := range result.Messages {
		if msg.Timestamp != mainMessageTS {
			replyMsg := models.LogMessage{
				Text:      msg.Text,
				User:      msg.User,
				Timestamp: msg.Timestamp,
			}
			replies = append(replies, replyMsg)
		}
	}

	return replies
}

func logMessage(msg models.LogMessage, userInfo models.UserInfo, channelID, channelName, msgType, parentMsgID string, replies map[string][]models.LogMessage) {
	if msgType == "main" {
		fmt.Printf("Message Type: %s, Channel: %s (ID: %s), User: %s (%s), Message: %s, UserID: %s, RealName: %s, Email: %s, Timestamp: %s\n",
			msgType, channelName, channelID, userInfo.Name, userInfo.ID, msg.Text, msg.User, userInfo.RealName, userInfo.Email, msg.Timestamp)
	} else if msgType == "reply" {
		fmt.Printf("Message Type: %s, Channel: %s (ID: %s), User: %s (%s), Message: %s, UserID: %s, RealName: %s, Email: %s, Timestamp: %s, ParentMessageID: %s\n",
			msgType, channelName, channelID, userInfo.Name, userInfo.ID, msg.Text, msg.User, userInfo.RealName, userInfo.Email, msg.Timestamp, parentMsgID)
	}
}

func GetUserInfo(userID string) (models.UserInfo, error) {
	params := map[string]string{
		"user": userID,
	}

	data, err := Fetch("users.info", params)
	if err != nil {
		return models.UserInfo{}, err
	}

	var result struct {
		Ok   bool `json:"ok"`
		User struct {
			ID       string `json:"id"`
			RealName string `json:"real_name"`
			Name     string `json:"name"`
			Profile  struct {
				Email string `json:"email"`
			} `json:"profile"`
		} `json:"user"`
	}

	if err := json.Unmarshal(data, &result); err != nil || !result.Ok {
		return models.UserInfo{}, fmt.Errorf("failed to fetch user info for user %s: %v", userID, err)
	}

	return models.UserInfo{
		ID:       result.User.ID,
		Name:     result.User.Name,
		RealName: result.User.RealName,
		Email:    result.User.Profile.Email,
	}, nil
}

func JoinChannel(channelID string) error {
	isMember, err := checkBotMembership(channelID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %v", err)
	}

	if !isMember {
		params := map[string]string{
			"channel": channelID,
		}

		data, err := Fetch("conversations.join", params)
		if err != nil {
			return fmt.Errorf("failed to join channel %s: %v", channelID, err)
		}

		var result ChannelResponse
		if err := json.Unmarshal(data, &result); err != nil || !result.Ok {
			return fmt.Errorf("failed to join channel: %v", err)
		}
	}

	return nil
}

func checkBotMembership(channelID string) (bool, error) {
	params := map[string]string{
		"channel": channelID,
	}

	data, err := Fetch("conversations.members", params)
	if err != nil {
		return false, fmt.Errorf("failed to fetch channel members: %v", err)
	}

	var result ChannelResponse
	if err := json.Unmarshal(data, &result); err != nil || !result.Ok {
		return false, fmt.Errorf("failed to parse member list: %v", err)
	}

	botUserID := os.Getenv("SLACK_BOT_USER_ID")
	for _, member := range result.Members {
		if member == botUserID {
			return true, nil
		}
	}

	return false, nil
}

func FetchChannels() ([]struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}, error) {
	params := map[string]string{
		"limit": "1000",
	}

	data, err := Fetch("conversations.list", params)
	if err != nil {
		return nil, err
	}

	var result ChannelResponse
	if err := json.Unmarshal(data, &result); err != nil || !result.Ok {
		return nil, fmt.Errorf("failed to fetch channels: %v", err)
	}

	return result.Channels, nil
}
