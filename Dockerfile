FROM golang:1.24.5-alpine

COPY . .

RUN go build 

CMD ./go-survey-bot
