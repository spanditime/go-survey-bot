module github.com/spanditime/go-survey-bot/vk

replace github.com/spanditime/go-survey-bot/conversation => ../../conversation

go 1.24.5

require (
	github.com/SevereCloud/vksdk/v3 v3.2.0
	github.com/spanditime/go-survey-bot/conversation v0.0.0-00010101000000-000000000000
)

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/text v0.23.0 // indirect
)
