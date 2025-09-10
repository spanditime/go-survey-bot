package vk

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/SevereCloud/vksdk/v3/api"
	longpoll "github.com/SevereCloud/vksdk/v3/longpoll-bot"
	"github.com/SevereCloud/vksdk/v3/events"
	"github.com/SevereCloud/vksdk/v3/object"
	"github.com/spanditime/go-survey-bot/conversation"
)

// Agent implementation vk via Long Poll

type Agent struct {
	vk *api.VK
	lp *longpoll.LongPoll
}

func NewBot(token string) (conversation.Agent, error) {
	if token == "" {
		return nil, fmt.Errorf("vk token is empty")
	}
	vk := api.NewVK(token)
	if vk == nil {
		return nil, fmt.Errorf("vk api initialization failed")
	}
	lp, err := longpoll.NewLongPollCommunity(vk)
	if err != nil {
		return nil, err
	}
	if lp == nil {
		return nil, fmt.Errorf("vk longpoll initialization failed")
	}
	return &Agent{
		vk: vk,
		lp: lp,
	}, nil
}

func (a *Agent) Run() (chan conversation.Update, error) {
	if a == nil || a.vk == nil || a.lp == nil {
		return nil, fmt.Errorf("vk agent is not initialized")
	}
	updates := make(chan conversation.Update)

	a.lp.MessageNew(func(ctx context.Context, obj events.MessageNewObject) {
		if updates != nil {
			updates <- newUpdate(a.vk, obj)
		}
	})

	go func() { _ = a.lp.Run() }()

	return updates, nil
}

type Update struct {
	vk  *api.VK
	obj events.MessageNewObject
}

func newUpdate(vk *api.VK, obj events.MessageNewObject) *Update {
	return &Update{vk: vk, obj: obj}
}

func (upd *Update) Provider() string { return "vk" }

func (upd *Update) ChatID() string {
	if upd == nil {
		return "vk"
	}
	if upd.obj.Message.PeerID != 0 {
		return fmt.Sprint("vk", upd.obj.Message.PeerID)
	}
	return fmt.Sprint("vk", upd.obj.Message.FromID)
}

func (upd *Update) GetSender() conversation.User {
	if upd == nil {
		return conversation.User{}
	}
	fromID := upd.obj.Message.FromID
	user := conversation.User{ Id: fmt.Sprint("vk", fromID) }
	if fromID > 0 && upd.vk != nil {
		if users, err := upd.vk.UsersGet(api.Params{"user_ids": strconv.Itoa(fromID)}); err == nil && len(users) > 0 {
			user.Name = users[0].FirstName
			user.Surname = users[0].LastName
		}
	}
	return user
}

func (upd *Update) GetMessage() string { return upd.obj.Message.Text }

func (upd *Update) Reply(text string) error {
	if upd == nil || upd.vk == nil {
		return fmt.Errorf("vk update/api is nil")
	}
	peerID := upd.obj.Message.PeerID
	if peerID == 0 {
		return fmt.Errorf("vk peer_id is 0")
	}
	_, err := upd.vk.MessagesSend(api.Params{
		"peer_id":   peerID,
		"message":   text,
		"random_id": int(time.Now().UnixNano() & 0x7fffffff),
	})
	return err
}

func (upd *Update) ReplyWithKeyboard(text string, kb []string) error {
	if upd == nil || upd.vk == nil {
		return fmt.Errorf("vk update/api is nil")
	}
	peerID := upd.obj.Message.PeerID
	if peerID == 0 {
		return fmt.Errorf("vk peer_id is 0")
	}
	if len(kb) == 0 {
		return upd.Reply(text)
	}
	keyboard := object.NewKeyboard(true)
	for _, label := range kb {
		keyboard.AddRow()
		keyboard.AddTextButton(label, "", "secondary")
	}
	_, err := upd.vk.MessagesSend(api.Params{
		"peer_id":   peerID,
		"message":   text,
		"random_id": int(time.Now().UnixNano() & 0x7fffffff),
		"keyboard":  keyboard,
	})
	return err
}


