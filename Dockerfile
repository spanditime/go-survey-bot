FROM golang:1.23.3-alpine

ADD *.go ./
ADD go.mod ./
ADD go.sum ./
RUN go build 
CMD ./go-survey-bot
