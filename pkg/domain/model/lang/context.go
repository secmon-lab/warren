package lang

import "context"

type ctxKey struct{}

func With(ctx context.Context, lang Lang) context.Context {
	return context.WithValue(ctx, ctxKey{}, lang)
}

func From(ctx context.Context) Lang {
	lang, ok := ctx.Value(ctxKey{}).(Lang)
	if !ok {
		return Default
	}
	return lang
}
