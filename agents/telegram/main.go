package tg

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spanditime/go-survey-bot/conversation"
)

// Agent implementation tg

type Agent struct {
	api *tgbotapi.BotAPI
}

func NewBot(token string) (conversation.Agent, error) {
	botapi, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Agent{
		api: botapi,
	}, err
}

func (tg *Agent) Run() (chan conversation.Update, error) {
	u := tgbotapi.NewUpdate(0)
	tg_updates := tg.api.GetUpdatesChan(u)
	updates := make(chan conversation.Update)
	go func() {
		for tg_update := range tg_updates {
			update := newUpdate(tg.api, tg_update)
			updates <- update
		}
	}()
	return updates, nil
}

type Update struct {
	api    *tgbotapi.BotAPI
	update tgbotapi.Update
}

func newUpdate(api *tgbotapi.BotAPI, tg_update tgbotapi.Update) *Update {
	return &Update{
		api:    api,
		update: tg_update,
	}
}

func (upd *Update) Provider() string {
	return "tg"
}
func (upd *Update) ChatID() string {
	ch := upd.update.FromChat()
	if ch != nil {
		return fmt.Sprint("tg", ch.ID)
	}
	return "tg" // todo: thats technically an error
}
func (upd *Update) GetSender() conversation.User {
	sent_from := upd.update.SentFrom()
	var name, surname, username, id string
	if sent_from != nil {
		name, surname, username, id = sent_from.FirstName, sent_from.LastName, sent_from.UserName, fmt.Sprint("tg", sent_from.ID)
	}
	return conversation.User{
		Name:     name,
		Surname:  surname,
		Id:       id,
		UserName: username,
	}
}
func (upd *Update) GetMessage() string {
	msg := upd.update.Message
	if msg != nil {
		return msg.Text
	}
	return ""
}
func (upd *Update) Reply(text string) error {
	reply_to := upd.update.Message
	if reply_to != nil {
		msg := tgbotapi.NewMessage(reply_to.Chat.ID, text)
		_, err := upd.api.Send(msg)
		if err != nil {
			return err
		}
	}
	//todo log an error here)
	return nil
}
func (upd *Update) ReplyWithKeyboard(text string, kb []string) error {
	reply_to := upd.update.Message
	if reply_to != nil {
		var buttons [][]tgbotapi.KeyboardButton = make([][]tgbotapi.KeyboardButton, len(kb))
		for i, b := range kb {
			buttons[i] = []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(b)}
		}
		keyb := tgbotapi.NewReplyKeyboard(buttons...)
		msg := tgbotapi.NewMessage(reply_to.Chat.ID, text)
		msg.ReplyMarkup = keyb
		_, err := upd.api.Send(msg)
		if err != nil {
			return err
		}
	}
	//todo log an error here)
	return nil
}
