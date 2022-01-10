package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tydar/mdbssg/models"
)

type postResponse struct {
	Title    string
	Subtitle string
	Author   string
	Content  []string
	Pubdate  string
}

// creates a postResponse object from a models.Post
func postResponseFromPostModel(post models.Post) postResponse {
	tokContent := strings.Split(strings.ReplaceAll(post.Content, "\r\n", "\n"), "\n\n")
	pd := post.Pubdate.Format("2006-01-02")
	return postResponse{
		Title:    post.Title,
		Subtitle: post.Subtitle,
		Author:   post.Author,
		Content:  tokContent,
		Pubdate:  pd,
	}
}

// ViewPost handles a server-side rendered post for pre-generation review
func (env *Env) ViewPost(w http.ResponseWriter, r *http.Request, au AuthUser) {
	slug := r.URL.Path[len("/post/"):] // get the part after /post/ with slicing
	if len(slug) > 0 {
		// if we have a slug, we pull the post and generate it
		// don't think we need to filter by username here
		post, err := env.posts.GetBySlug(r.Context(), slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// need to find a cleaner way to document typing for these structs
		// actually what we want to do is create a PostResponse type
		// and reformat the models.Post.Content -> []string split on newlines
		// so that we can change the template to wrap each split into <p>...</p>
		td := struct {
			Post     postResponse
			LoggedIn bool
			Flash    string
		}{
			Post:     postResponseFromPostModel(post),
			LoggedIn: false,
			Flash:    "",
		}

		err = env.templates["view_post"].ExecuteTemplate(w, "base", td)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// had no slug after the URL
		// we want a list of posts that belong to the logged-in user
	}
}

//EditPost handles GET requests to render the edit form for a post
func (env *Env) EditPost(w http.ResponseWriter, r *http.Request, au AuthUser) {
	slug := r.URL.Path[len("/edit/"):]
	post, err := env.posts.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if au.user.Username != post.OwnerUsername {
		http.Error(w, "not authorized: wrong user", http.StatusUnauthorized)
		return
	}

	// pull out some specific fields from the models.Post
	// because postResponse doesn't give what we need
	td := struct {
		Slug     string
		Post     postResponse
		Content  string
		LoggedIn bool
		Flash    string
	}{
		Slug:     post.Slug,
		Content:  post.Content,
		Post:     postResponseFromPostModel(post),
		LoggedIn: true,
		Flash:    "",
	}

	err = env.templates["edit_post"].ExecuteTemplate(w, "base", td)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// form to create a new post
// uses the same template as EditPost but with empty post struct
// generates slug from title-pubdate
func (env *Env) NewPost(w http.ResponseWriter, r *http.Request, au AuthUser) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		subtitle := r.FormValue("subtitle")
		author := r.FormValue("author")
		date := r.FormValue("date")
		username := au.user.Username
		content := r.FormValue("content")

		dateParseString := "2006-01-02"
		pubdate, err := time.Parse(dateParseString, date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		slug := filepath.Clean(title + "-" + date)

		post := models.Post{
			Title:         title,
			Subtitle:      subtitle,
			Author:        author,
			Pubdate:       pubdate,
			OwnerUsername: username,
			Content:       content,
			Slug:          slug,
		}

		_, err = env.posts.GetBySlug(r.Context(), slug)
		if err == nil {
			// render the new post form with an error flash
			// we already have a post with this slug
			td := struct {
				Post     models.Post
				LoggedIn bool
				Flash    string
			}{
				Post:     post,
				LoggedIn: true,
				Flash:    "post with this title and publish date already exists",
			}
			err := env.templates["new_post"].ExecuteTemplate(w, "base", td)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		err = env.posts.Create(r.Context(), post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		http.Redirect(w, r, "/post/"+slug, http.StatusFound)
		return
	} else if r.Method == "GET" {
		td := struct {
			Post     models.Post
			LoggedIn bool
			Flash    string
		}{
			Post:     models.Post{},
			LoggedIn: true,
			Flash:    "",
		}
		err := env.templates["new_post"].ExecuteTemplate(w, "base", td)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	return
}

// SavePost handles POST requests to update or create a post
func (env *Env) SavePost(w http.ResponseWriter, r *http.Request, au AuthUser) {
	slug := r.URL.Path[len("/save/"):]
	title := r.FormValue("title")
	subtitle := r.FormValue("subtitle")
	author := r.FormValue("author")
	date := r.FormValue("date")
	username := au.user.Username
	content := r.FormValue("content")

	dateParseString := "2006-01-02"
	pubdate, err := time.Parse(dateParseString, date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	post := models.Post{
		Title:         title,
		Subtitle:      subtitle,
		Author:        author,
		Pubdate:       pubdate,
		OwnerUsername: username,
		Content:       content,
		Slug:          slug,
	}

	post, err = env.posts.GetBySlug(r.Context(), slug)
	if err == nil && post.OwnerUsername == username {
		fmt.Println("updating")
		err := env.posts.Update(r.Context(), post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if post.OwnerUsername != username {
		http.Error(w, "auth: incorrect user", http.StatusUnauthorized)
		return
	} else {
		fmt.Println("creating")
		err := env.posts.Create(r.Context(), post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/post/"+slug, http.StatusFound)
}

// GeneratePosts invokes functions to generate a new static site
// at the configured subdir
func (env *Env) GeneratePosts(w http.ResponseWriter, r *http.Request, au AuthUser) {
	username := au.user.Username
	posts, err := env.posts.GetByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, p := range posts {
		pr := postResponseFromPostModel(p)
		text, err := env.generatePost(pr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = saveGeneratedPost(text, p.Slug, "output", username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/static/"+username+"/", http.StatusFound)
}

func (env *Env) generatePost(pr postResponse) (string, error) {
	t := env.templates["gen_post"]
	buf := new(bytes.Buffer)
	td := struct {
		Post postResponse
	}{
		Post: pr,
	}
	err := t.ExecuteTemplate(buf, "base", td)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func saveGeneratedPost(text, slug, dir, username string) error {
	path := filepath.Join(".", "static", username)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(path, slug) + ".html")
	if err != nil {
		return err
	}

	f.WriteString(text)
	f.Close()
	return nil
}
