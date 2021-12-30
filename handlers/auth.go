package handlers

import (
	"errors"
	"net/http"

	"github.com/tydar/mdbssg/models"
)

type AuthenticatedHandler func(w http.ResponseWriter, r *http.Request, authUser AuthUser)

type AuthMW struct {
	handler AuthenticatedHandler
	e       *Env
}

type AuthUser struct {
	user  models.User
	token string
}

func NewAuthMW(h AuthenticatedHandler, e *Env) *AuthMW {
	return &AuthMW{
		handler: h,
		e:       e,
	}
}

func (a *AuthMW) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, token, err := a.e.getSignedInUser(r)
	if err != nil {
		http.Error(w, "please sign-in", http.StatusUnauthorized)
		return
	}

	au := AuthUser{
		user:  user,
		token: token,
	}

	a.handler(w, r, au)
}

func (e *Env) getSignedInUser(r *http.Request) (models.User, string, error) {
	username, err := r.Cookie("user")
	if err != nil {
		return models.User{}, "", err
	}

	token, err := r.Cookie("sessionid")
	if err != nil {
		return models.User{}, "", err
	}

	if !e.users.CheckSessionValid(r.Context(), username.Value, token.Value) {
		return models.User{}, "", errors.New("no valid session")
	}

	user, err := e.users.GetByUsername(r.Context(), username.Value)
	return user, token.Value, err
}

func (e *Env) checkSignedIn(r *http.Request) error {
	username, err := r.Cookie("user")
	if err != nil {
		return err
	}

	token, err := r.Cookie("sessionid")
	if err != nil {
		return err
	}

	if !e.users.CheckSessionValid(r.Context(), username.Value, token.Value) {
		return errors.New("no valid session")
	}
	return nil
}
