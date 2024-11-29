package rabbitMQ

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"xyz-task-5/internal/db"
	"xyz-task-5/internal/models"
	"xyz-task-5/internal/slack"

	"github.com/streadway/amqp"
)

type QueueMessage struct {
	Type          string            `json:"type"`
	ChannelID     string            `json:"channel_id"`
	ChannelName   string            `json:"channel_name"`
	Message       models.LogMessage `json:"message"`
	UserInfo      models.UserInfo   `json:"user_info"`
	MainMessageID int               `json:"main_message_id,omitempty"`
}

var (
	conn    *amqp.Connection
	channel *amqp.Channel
)

func InitRabbitMQ() error {

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		return fmt.Errorf("RABBITMQ_URL not set")
	}

	var err error
	conn, err = amqp.Dial(rabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	channel, err = conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	_, err = channel.QueueDeclare(
		"user_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare user queue: %v", err)
	}

	_, err = channel.QueueDeclare(
		"channel_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare channel queue: %v", err)
	}

	_, err = channel.QueueDeclare(
		"message_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare message queue: %v", err)
	}

	return nil
}

func PublishMessage(queueName string, msg QueueMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	log.Printf("Publishing message to RABBITMQ queue %s: %s", queueName, string(body))

	return channel.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func StartUserWorker() {
	msgs, err := channel.Consume(
		"user_queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	for msg := range msgs {
		log.Printf("Received message from user_queue: %s", string(msg.Body))
		var queueMsg QueueMessage
		err := json.Unmarshal(msg.Body, &queueMsg)
		if err != nil {
			log.Printf("Error unmarshaling user message: %v", err)
			msg.Nack(false, false)
			continue
		}

		err = db.InsertUser(queueMsg.UserInfo)
		if err != nil {
			log.Printf("inserting user[not] : %v", err)
			msg.Nack(false, true)
		} else {
			msg.Ack(false)
		}
	}
}

func StartChannelWorker() {
	msgs, err := channel.Consume(
		"channel_queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	for msg := range msgs {
		var queueMsg QueueMessage
		err := json.Unmarshal(msg.Body, &queueMsg)
		if err != nil {
			log.Printf("Error unmarshaling channel message: %v", err)
			msg.Nack(false, false)
			continue
		}

		channel := struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   queueMsg.ChannelID,
			Name: queueMsg.ChannelName,
		}

		err = db.InsertChannel(channel)
		if err != nil {
			log.Printf("inserting channel[not]: %v", err)
			msg.Nack(false, true)
		} else {
			msg.Ack(false)
		}
	}
}

func StartMessageWorker() {
	msgs, err := channel.Consume(
		"message_queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	for msg := range msgs {
		var queueMsg QueueMessage
		err := json.Unmarshal(msg.Body, &queueMsg)
		if err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			msg.Nack(false, false)
			continue
		}

		if queueMsg.Type == "main_message" {

			mainMessageID, err := db.InsertMainMessage(queueMsg.Message, queueMsg.ChannelID, queueMsg.UserInfo.ID)
			if err != nil {
				log.Printf("inserting main message[not]: %v", err)
				msg.Nack(false, true)
			} else {
				msg.Ack(false)

				replies := slack.FetchAllReplies(queueMsg.ChannelID, queueMsg.ChannelName, queueMsg.Message.Timestamp)
				for _, reply := range replies {
					replyUserInfo, err := slack.GetUserInfo(reply.User)
					if err != nil {
						continue
					}

					replyMsg := QueueMessage{
						Type:          "reply_message",
						ChannelID:     queueMsg.ChannelID,
						ChannelName:   queueMsg.ChannelName,
						Message:       reply,
						UserInfo:      replyUserInfo,
						MainMessageID: mainMessageID,
					}
					err = PublishMessage("message_queue", replyMsg)
					if err != nil {
						log.Printf("Error publishing reply message: %v", err)
					}
				}
			}
		} else if queueMsg.Type == "reply_message" {

			err := db.InsertReplyMessage(queueMsg.Message, queueMsg.MainMessageID, queueMsg.ChannelID, queueMsg.UserInfo.ID)
			if err != nil {
				log.Printf("inserting reply message[not]: %v", err)
				msg.Nack(false, true)
			} else {
				msg.Ack(false)
			}
		}
	}
}
