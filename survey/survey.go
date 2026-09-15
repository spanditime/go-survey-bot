package survey

import (
	"fmt"
	"log"
	"time"

	"github.com/spanditime/go-survey-bot/consts"
	"github.com/spanditime/go-survey-bot/conversation"
)

func newYesNoConversationHandler(question string, welcome conversation.Action, no conversation.Action, yes conversation.Action, cancel conversation.Action) *conversation.OptionsHandler {
	handlers := conversation.OptionsHandlers{
		consts.Yes: yes,
		consts.No:  no,
	}
	handlers[consts.Cancel] = cancel
	return conversation.NewOptionsHandler(welcome, question, handlers, conversation.EmptyAction())
}

type SurveyFabric struct {
	db *SurveyDB
}

func (f *SurveyFabric) NewStartQuestion() conversation.Handler {
	handle := func(answer string, ctx conversation.Ctx) error {
		if answer == "/start" {
			return conversation.TransitionStageAction(f.newWelcomeQuestion)(answer, ctx)
		}
		return nil
	}
	cancel := conversation.TransitionStageAction(f.NewStartQuestion)
	handlers := conversation.OptionsHandlers{
		"/start": handle,
	}
	return conversation.NewOptionsHandler(conversation.EmptyAction(), consts.StartMessage, handlers, cancel)
}

func (f *SurveyFabric) newWelcomeQuestion() conversation.Handler {
	next := conversation.TransitionStageActionCtx(f.newNameQuestion(false))
	cancel := conversation.TransitionStageAction(f.NewStartQuestion)
	return newYesNoConversationHandler(consts.GoToSurvey, conversation.SendTextAction(consts.WelcomeMessage, conversation.EmptyAction()), cancel, next, cancel)
}

func saveSurveyAnswer(key string, fall bool, save conversation.Action, next conversation.Action) conversation.Action {
	var action conversation.Action
	if fall {
		action = conversation.SaveKeyAction(key, save)
	} else {
		action = conversation.SaveKeyAction(key, next)
	}
	return action
}

func (f *SurveyFabric) newNameQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.NewStartQuestion)
		defaultName := ctx.Update().GetSender().FullName()
		save := saveSurveyAnswer(consts.NameKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newAgeQuestion(false)))
		handlers := conversation.OptionsHandlers{
			consts.Cancel:      cancel,
			defaultName: save,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), consts.EnterName, handlers, save)
	}
}

func (f *SurveyFabric) newAgeQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.NewStartQuestion)
		save := saveSurveyAnswer(consts.AgeKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newCityQuestion(false)))
		handlers := conversation.OptionsHandlers{
			consts.Cancel: cancel,
		}
		// todo: if have age - add age
		if false {
			defaultAge := "20"
			handlers[defaultAge] = save
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), consts.EnterAge, handlers, save)
	}
}

func (f *SurveyFabric) newCityQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.NewStartQuestion)
		save := saveSurveyAnswer(consts.CityKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newRequestQuestion(false)))
		handlers := conversation.OptionsHandlers{
			consts.Cancel: cancel,
			consts.Yes:    save,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), consts.EnterCity, handlers, save)
	}
}

func (f *SurveyFabric) newRequestQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.NewStartQuestion)
		save := saveSurveyAnswer(consts.RequestKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newHealthQuestion(false)))
		handlers := conversation.OptionsHandlers{
			consts.Cancel: cancel,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), consts.EnterRequest, handlers, save)
	}
}

func (f *SurveyFabric) newHealthQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.NewStartQuestion)
		save := saveSurveyAnswer(consts.HealthKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newContactQuestion))
		handlers := conversation.OptionsHandlers{
			consts.Cancel: cancel,
			consts.Yes:    save,
			consts.No:     save,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), consts.EnterHealth, handlers, save)
	}
}

func (f *SurveyFabric) newContactQuestion(answer string, ctx conversation.Ctx) conversation.Handler {
	cancel := conversation.TransitionStageAction(f.NewStartQuestion)
	save := conversation.SaveKeyAction(consts.ContactKey, conversation.TransitionStageActionCtx(f.newSaveQuestion))
	handlers := conversation.OptionsHandlers{
		consts.Cancel: cancel,
	}
	// todo: if have contact - add it
	username := ctx.Update().GetSender().UserName
	if len(username) > 0 {
		defaultContact := fmt.Sprint(ctx.Update().Provider(), ": ", username)
		handlers[defaultContact] = save
	}
	return conversation.NewOptionsHandler(conversation.EmptyAction(), consts.EnterContact, handlers, save)
}

func (f *SurveyFabric) newSaveQuestion(answer string, ctx conversation.Ctx) conversation.Handler {
	name, _ := ctx.GetKey(consts.NameKey)
	age, _ := ctx.GetKey(consts.AgeKey)
	city, _ := ctx.GetKey(consts.CityKey)
	request, _ := ctx.GetKey(consts.RequestKey)
	health, _ := ctx.GetKey(consts.HealthKey)
	contact, _ := ctx.GetKey(consts.ContactKey)
	question := fmt.Sprintf("%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		consts.EnterName, name,
		consts.EnterAge, age,
		consts.EnterCity, city,
		consts.EnterRequest, request,
		consts.EnterHealth, health,
		consts.EnterContact, contact,
		consts.Accept)
	saveSurvey := func(answer string, ctx conversation.Ctx) error {
		if err := conversation.SendTextAction(consts.Thanks, conversation.EmptyAction())(answer, ctx); err != nil {
			return err
		}

		id := ctx.Update().ChatID()
		contact = fmt.Sprintf("%s (%s: %s)", contact, ctx.Update().Provider(), ctx.Update().GetSender().UserName)
		err := f.db.WriteAnswers(
			id,
			time.Now(),
			name,
			age,
			city,
			request,
			health,
			contact,
		)
		if err != nil {
			// todo: log an error
			log.Printf("Cant write survey results for user %s %s: %v", id, contact, err)
			// todo: notify user
		}
		return conversation.TransitionStageAction(f.NewStartQuestion)(answer, ctx)
	}
	return conversation.NewOptionsHandler(conversation.EmptyAction(), question, conversation.OptionsHandlers{
		consts.Submit:        saveSurvey,
		consts.ChangeName:    conversation.TransitionStageActionCtx(f.newNameQuestion(true)),
		consts.ChangeAge:     conversation.TransitionStageActionCtx(f.newAgeQuestion(true)),
		consts.ChangeCity:    conversation.TransitionStageActionCtx(f.newCityQuestion(true)),
		consts.ChangeRequest: conversation.TransitionStageActionCtx(f.newRequestQuestion(true)),
		consts.ChangeHealth:  conversation.TransitionStageActionCtx(f.newHealthQuestion(true)),
		consts.ChangeContact: conversation.TransitionStageActionCtx(f.newContactQuestion),
		consts.Cancel: func(answer string, ctx conversation.Ctx) error {
			// note: clear context storage might be needed
			return conversation.TransitionStageAction(f.NewStartQuestion)(answer, ctx)
		},
	}, conversation.EmptyAction())
}

func NewSurveyFabric(db *SurveyDB) *SurveyFabric {
	return &SurveyFabric{
		db: db,
	}
}
