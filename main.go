package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tydar/mdbssg/handlers"
	"github.com/tydar/mdbssg/host"
	"github.com/tydar/mdbssg/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"cloud.google.com/go/storage"
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

	bucket, prs := os.LookupEnv("BUCKET")
	if !prs {
		log.Fatal("no google storage name in $BUCKET")
	}

	_, prs = os.LookupEnv("HEROKU")
	if prs {
		// we need to get our creds from the environment and write them to the disk so it works
		creds := os.Getenv("GOOGLE_CREDENTIALS")
		f, err := os.Create("/app/gcp-credentials.json")
		if err != nil {
			panic(err)
		}
		_, err = f.Write([]byte(creds))
		if err != nil {
			panic(err)
		}
		f.Close()
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
	gsClient, err := storage.NewClient(context.Background())
	if err != nil {
		panic(err)
	}

	theHost := host.NewGSHost(bucket, gsClient)
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
