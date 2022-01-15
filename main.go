package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/tydar/mdbssg/handlers"
	"github.com/tydar/mdbssg/host"
	"github.com/tydar/mdbssg/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dbUrl, prs := os.LookupEnv("DATABASE_URL")
	if !prs {
		dbUrl = "mongodb://localhost:27017"
	}

	port, prs := os.LookupEnv("PORT")
	if !prs {
		port = "8080"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbUrl))
	if err != nil {
		panic(err)
	}

	um := models.NewUserModel(client, "mdbssg")
	pm := models.NewPostModel(client, "mdbssg")

	t := map[string]*template.Template{"signin": template.Must(template.ParseFiles("templates/base.html", "templates/signin.html"))}
	t["changepwd"] = template.Must(template.ParseFiles("templates/base.html", "templates/changepwd.html"))
	t["signup"] = template.Must(template.ParseFiles("templates/base.html", "templates/signup.html"))
	t["view_post"] = template.Must(template.ParseFiles("templates/base.html", "templates/post.html"))
	t["edit_post"] = template.Must(template.ParseFiles("templates/base.html", "templates/edit_post.html"))
	t["gen_post"] = template.Must(template.ParseFiles("templates/base_gen.html", "templates/post.html"))
	t["new_post"] = template.Must(template.ParseFiles("templates/base.html", "templates/new_post.html"))
	t["list_posts"] = template.Must(template.ParseFiles("templates/base.html", "templates/posts.html"))

	//theHost := host.NewLocalHost("static")
	theHost := host.NewGSHost("mdbssg-test-bucket")
	env := handlers.NewEnv(um, pm, t, theHost)

	http.HandleFunc("/", handlers.NewAuthMW(env.ViewPost, env).ServeHTTP)
	http.HandleFunc("/signin/", env.SignIn)
	http.HandleFunc("/changepwd/", handlers.NewAuthMW(env.ChangePassword, env).ServeHTTP)
	http.HandleFunc("/signup/", env.SignUpHandler)
	http.HandleFunc("/signout/", handlers.NewAuthMW(env.SignOut, env).ServeHTTP)
	http.HandleFunc("/post/", handlers.NewAuthMW(env.ViewPost, env).ServeHTTP)
	http.HandleFunc("/edit/", handlers.NewAuthMW(env.EditPost, env).ServeHTTP)
	http.HandleFunc("/save/", handlers.NewAuthMW(env.SavePost, env).ServeHTTP)
	http.HandleFunc("/generate/", handlers.NewAuthMW(env.GeneratePosts, env).ServeHTTP)
	http.HandleFunc("/new/", handlers.NewAuthMW(env.NewPost, env).ServeHTTP)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
