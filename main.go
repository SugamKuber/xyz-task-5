package main

import (
	"fmt"
	"log"

	"sync"

	"github.com/joho/godotenv"

	"xyz-task-5/internal/db"
	"xyz-task-5/internal/rabbitMQ"
	"xyz-task-5/internal/slack"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	db.InitDB()

	slack.InitSlackClient()

	err = rabbitMQ.InitRabbitMQ()
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		rabbitMQ.StartUserWorker()
	}()
	go func() {
		defer wg.Done()
		rabbitMQ.StartChannelWorker()
	}()
	go func() {
		defer wg.Done()
		rabbitMQ.StartMessageWorker()
	}()

	channels, err := slack.FetchChannels()
	if err != nil {
		log.Fatalf("Error fetching channels: %v", err)
	}

	for _, ch := range channels {

		err := slack.JoinChannel(ch.ID)
		if err != nil {
			log.Printf("Error joining channel %s: %v", ch.ID, err)
			continue
		}

		channelMsg := rabbitMQ.QueueMessage{
			Type:        "channel",
			ChannelID:   ch.ID,
			ChannelName: ch.Name,
		}
		err = rabbitMQ.PublishMessage("channel_queue", channelMsg)
		if err != nil {
			log.Printf("Error publishing channel: %v", err)
			continue
		}

		messages, err := slack.FetchChannelMessages(ch.ID, ch.Name)
		if err != nil {
			log.Printf("Error fetching messages for channel %s: %v", ch.Name, err)
			continue
		}

		for _, msg := range messages {
			userInfo, err := slack.GetUserInfo(msg.User)
			if err != nil {
				continue
			}

			userMsg := rabbitMQ.QueueMessage{
				Type:     "user",
				UserInfo: userInfo,
			}
			err = rabbitMQ.PublishMessage("user_queue", userMsg)
			if err != nil {
				log.Printf("Error publishing user: %v", err)
				continue
			}

			messageMsg := rabbitMQ.QueueMessage{
				Type:        "main_message",
				ChannelID:   ch.ID,
				ChannelName: ch.Name,
				Message:     msg,
				UserInfo:    userInfo,
			}
			err = rabbitMQ.PublishMessage("message_queue", messageMsg)
			if err != nil {
				log.Printf("Error publishing message: %v", err)
			}
		}
	}

	wg.Wait()

	fmt.Println("All messages processed through RabbitMQ queue.")
}
