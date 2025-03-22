package service

/*
type ActionService struct {
	actions map[string]interfaces.Action
}

func NewActionService(actions []interfaces.Action) *ActionService {
	actionsMap := make(map[string]interfaces.Action)
	for _, a := range actions {
		if a.Spec().Name == "" {
			panic(goerr.New("action name is required", goerr.V("action", a)))
		}
		if _, ok := actionsMap[a.Spec().Name]; ok {
			panic(goerr.New("action name is duplicated", goerr.V("action", a)))
		}
		actionsMap[a.Spec().Name] = a
	}

	return &ActionService{actions: actionsMap}
}

func (x *ActionService) Spec() []action.ActionSpec {
	specs := make([]action.ActionSpec, 0, len(x.actions))
	for _, a := range x.actions {
		specs = append(specs, a.Spec())
	}
	return specs
}

func (x *ActionService) Execute(ctx context.Context, slack interfaces.SlackThreadService, name string, ssn interfaces.LLMSession, args action.Arguments) (*action.ActionResult, error) {
	logger := logging.From(ctx)
	action, ok := x.actions[name]
	if !ok {
		return nil, goerr.New("unknown action", goerr.V("name", name))
	}

	logger.Info("executing action", "name", name, "args", args)
	return action.Execute(ctx, slack, ssn, args)
}
*/
