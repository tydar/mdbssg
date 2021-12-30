package handlers

import "context"

type TemplateData struct {
	Flash    string
	LoggedIn bool
}

func (td *TemplateData) String() string {
	return td.Flash
}

type flashKey string

const fk flashKey = "flash"

func NewContextTD(ctx context.Context, td *TemplateData) context.Context {
	return context.WithValue(ctx, fk, td)
}

func tdFromContext(ctx context.Context) (*TemplateData, bool) {
	td, ok := ctx.Value(fk).(*TemplateData)
	return td, ok
}
