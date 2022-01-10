package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/tydar/mdbssg/models"
	"go.mongodb.org/mongo-driver/mongo"
)

func (env *Env) SignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		err := env.checkSignedIn(r)
		fmt.Println(err)
		if err == nil {
			http.Redirect(w, r, "/changepwd/", http.StatusFound)
			return
		}
		err = env.templates["signin"].ExecuteTemplate(w, "base", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		err := env.checkSignedIn(r)
		if err == nil {
			http.Redirect(w, r, "/changepwd/", http.StatusFound)
			return
		}
		u, err := env.users.GetByUsername(r.Context(), r.FormValue("username"))
		if err == mongo.ErrNoDocuments {
			td := TemplateData{Flash: "No user found! Try again."}
			err = env.templates["signin"].ExecuteTemplate(w, "base", td)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		good := models.CheckPassword(u, r.FormValue("password"))
		if good {
			t, err := env.users.AppendNewSession(r.Context(), u.Username)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sessionCookie := http.Cookie{Name: "sessionid", Value: t, SameSite: 2, HttpOnly: true, Path: "/"}
			unameCookie := http.Cookie{Name: "user", Value: u.Username, SameSite: 2, HttpOnly: true, Path: "/"}

			http.SetCookie(w, &sessionCookie)
			http.SetCookie(w, &unameCookie)
			http.Redirect(w, r, "/changepwd/", http.StatusFound)
		} else {
			td := TemplateData{Flash: "Incorrect password! Please try again."}
			err := env.templates["signin"].ExecuteTemplate(w, "base", td)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}

func (env *Env) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	err := env.checkSignedIn(r)
	if err == nil {
		http.Error(w, "already logged in", http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		err = env.templates["signup"].ExecuteTemplate(w, "base", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		if r.FormValue("password") != r.FormValue("confirmpassword") {
			http.Error(w, "passwords do not match", http.StatusInternalServerError)
			return
		}

		err = env.users.CreateUser(r.Context(), r.FormValue("username"), r.FormValue("password"), r.FormValue("username"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// successfully signed up user, redirect to sign in form
		td := TemplateData{Flash: "Account successfully created!"}
		err := env.templates["signin"].ExecuteTemplate(w, "base", td)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (env *Env) SignOut(w http.ResponseWriter, r *http.Request, au AuthUser) {
	err := env.users.InvalidateSession(r.Context(), au.user.Username, au.token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/signin/", http.StatusFound)
	return
}

func (env *Env) ChangePassword(w http.ResponseWriter, r *http.Request, au AuthUser) {
	if r.Method == "GET" {
		td := TemplateData{LoggedIn: true}
		err := env.templates["changepwd"].ExecuteTemplate(w, "base", td)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if r.Method == "POST" {
		err := env.changePassword(r, au.user)
		if err != nil {
			td := TemplateData{Flash: err.Error(), LoggedIn: true}
			err = env.templates["changepwd"].ExecuteTemplate(w, "base", td)
		}
		td := TemplateData{Flash: "Password changed successfully!", LoggedIn: true}
		err = env.templates["changepwd"].ExecuteTemplate(w, "base", td)
	}
	return
}

func (env *Env) changePassword(r *http.Request, user models.User) error {
	oldPassword := r.FormValue("oldpassword")
	good := models.CheckPassword(user, oldPassword)
	if !good {
		return errors.New("Current password incorrect.")
	}

	good = r.FormValue("newpassword") == r.FormValue("confirmpassword")
	if !good {
		return errors.New("New password and confirmation value do not match.")
	}

	return env.users.UpdatePassword(r.Context(), user.Username, r.FormValue("newpassword"))
}
