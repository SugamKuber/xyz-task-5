# Docs

docker build -t slack-message-processor .

docker run -env-file .env slack-message-processor

docker compose down

docker compose up --build


To start application normally

go run main.go

```
On start of application, messages will be fetched from slack and inserted to db if not exist
```

Sample .env
```
SLACK_API_URI=https://slack.com/api/
SLACK_TOKEN=xoxb-***********
DB_URI=postgresql://***********
SLACK_BOT_USER_ID=***********
RABBITMQ_URL=amqp://admin:adminpassword@rabbitmq:5672/
```