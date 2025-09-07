FROM golang:1.24.5-alpine

ADD *.go ./
ADD go.mod ./
ADD go.sum ./
ADD conversation ./conversation
ADD agents ./agents
RUN go build 
CMD ./go-survey-bot
