FROM golang:1.24.5-alpine

COPY . .

CMD ./go-survey-bot
