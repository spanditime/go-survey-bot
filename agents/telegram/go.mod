module github.com/spanditime/go-survey-bot/telegram

replace github.com/spanditime/go-survey-bot/conversation => ../../conversation

go 1.24.5

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/spanditime/go-survey-bot/conversation v0.0.0-00010101000000-000000000000
)
