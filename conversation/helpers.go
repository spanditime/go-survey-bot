package conversation
// conversation handler with predefined options
type Action = func(text string, ctx Ctx) error
type Stage = func() Handler
type StageCtx = func(answer string, ctx Ctx) Handler

func EmptyAction() Action { return func(answer string, ctx Ctx) error { return nil } }
func SendTextAction(text string, next Action) Action {
	return func(answer string, ctx Ctx) error {
		ctx.Update().Reply(text)
		next(answer, ctx)
		return nil
	}
}
func SendTextWithKeyboardAction(text string, keyboard []string, next Action) Action {
	return func(answer string, ctx Ctx) error {
		ctx.Update().ReplyWithKeyboard(text, keyboard)
		next(answer, ctx)
		return nil
	}
}
func TransitionStageAction(next Stage) Action {
	return func(answer string, ctx Ctx) error {
		ctx.SetNext(next())
		return nil
	}
}
func TransitionStageActionCtx(next StageCtx) Action {
	return func(answer string, ctx Ctx) error {
		ctx.SetNext(next(answer, ctx))
		return nil
	}
}

func SaveKeyAction(key string, next Action) Action {
	return func(answer string, ctx Ctx) error {
		ctx.SetKey(key, answer)
		return next(answer, ctx)
	}
}

type AnswerHandler struct {
	welcome       Action
	question      string
	answerHandler Action
}

func NewAnswerHandler(welcome Action, question string, answerHandler Action) Handler {
	return &AnswerHandler{
		welcome:       welcome,
		question:      question,
		answerHandler: answerHandler,
	}
}

func (handler *AnswerHandler) sendQuestion(ctx Ctx) error {
	return ctx.Update().Reply(handler.question)
}
func (handler *AnswerHandler) Welcome(ctx Ctx) error {
	err := handler.welcome("", ctx)
	if err != nil {
		return err
	}
	return handler.sendQuestion(ctx)
}

func (handler *AnswerHandler) Handle(ctx Ctx) error {
	var answer string = ctx.Update().GetMessage()
	return handler.answerHandler(answer, ctx)
}

type OptionsHandlers map[string]Action

func (handlers *OptionsHandlers) Options() []string {
	keys := make([]string, len(*handlers))
	i := 0
	for k := range *handlers {
		keys[i] = k
		i++
	}
	return keys
}

type OptionsHandler struct {
	welcome        Action
	optionHandlers OptionsHandlers
	question       string
	answerHandler  Action
}

func NewOptionsHandler(welcome Action, question string, handlers OptionsHandlers, answerHandler Action) *OptionsHandler {
	return &OptionsHandler{
		welcome:        welcome,
		optionHandlers: handlers,
		question:       question,
		answerHandler:  answerHandler,
	}
}

func (handler *OptionsHandler) sendQuestion(ctx Ctx) error {
	return SendTextWithKeyboardAction(handler.question, handler.optionHandlers.Options(), EmptyAction())("", ctx)
}

func (handler *OptionsHandler) handleOption(option string, ctx Ctx) error {
	var err error
	if h, found := handler.optionHandlers[option]; found {
		err = h(option, ctx)
	} else {
		err = handler.answerHandler(option, ctx)
	}
	return err
}

func (handler *OptionsHandler) Handle(ctx Ctx) error {
	var answer string = ctx.Update().GetMessage()
	if err := handler.handleOption(answer, ctx); err != nil {
		return err
	}
	if !ctx.Finalized() && !ctx.Transitioning() {
		return handler.sendQuestion(ctx)
	}
	return nil
}

func (handler *OptionsHandler) Welcome(ctx Ctx) error {
	if err := handler.welcome("", ctx); err != nil {
		return err
	}
	return handler.sendQuestion(ctx)
}
