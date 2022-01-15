package handlers

import (
	"context"
	"html/template"

	"github.com/tydar/mdbssg/host"
	"github.com/tydar/mdbssg/models"
)

// Env wraps interfaces that describe the DB actions required by http handlers
// as well as template information.
// following the pattern from https://www.alexedwards.net/blog/organising-database-access
type Env struct {
	users     Users
	posts     Posts
	theHost   host.Host
	templates map[string]*template.Template
}

// Users interface describes the set of behaviors that need to be available for user record & session management
type Users interface {
	CreateUser(context context.Context, username, password, display string) error
	GetByUsername(context context.Context, username string) (models.User, error)
	AppendNewSession(context context.Context, username string) (string, error)
	CheckSessionValid(context context.Context, username, token string) bool
	UpdatePassword(context context.Context, username, password string) error
	InvalidateSession(context context.Context, username, token string) error
}

type Posts interface {
	GetBySlug(ctx context.Context, slug string) (models.Post, error)
	GetByUsername(ctx context.Context, username string) ([]models.Post, error)
	Create(ctx context.Context, post models.Post) error
	Update(ctx context.Context, post models.Post) error
}

func NewEnv(users Users, posts Posts, templates map[string]*template.Template, theHost host.Host) *Env {
	return &Env{
		users:     users,
		posts:     posts,
		templates: templates,
		theHost:   theHost,
	}
}
