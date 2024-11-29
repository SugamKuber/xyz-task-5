package models

type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Email    string `json:"email"`
}

type LogMessage struct {
	Text      string `json:"text"`
	User      string `json:"user"`
	Timestamp string `json:"ts"`
}
