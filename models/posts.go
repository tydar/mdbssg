package models

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PostModel struct {
	client *mongo.Client
	dbName string
}

func NewPostModel(client *mongo.Client, db string) *PostModel {
	return &PostModel{
		client: client,
		dbName: db,
	}
}

type PostAlreadyExists struct {
	slug string
}

func (e *PostAlreadyExists) Error() string {
	return "a post with the slug already exists: " + e.slug
}

// given a slug, look up and return the Post struct and a nil value for error
// otherwise, return the error provided by the mongo driver.
func (pm *PostModel) GetBySlug(ctx context.Context, slug string) (Post, error) {
	posts := pm.client.Database(pm.dbName).Collection("posts")

	sr := posts.FindOne(ctx, bson.M{"slug": slug})

	if sr.Err() != nil {
		return Post{}, sr.Err()
	}

	var post Post
	err := sr.Decode(&post)
	if err != nil {
		return Post{}, err
	}

	return post, nil
}

// given a Post struct, either create the post (if no post with this slug is found in the db)
// or return a PostAlreadyExists error or return a mongo error
func (pm *PostModel) Create(ctx context.Context, post Post) error {
	posts := pm.client.Database(pm.dbName).Collection("posts")

	_, err := pm.GetBySlug(ctx, post.Slug)
	if err == nil {
		return &PostAlreadyExists{slug: post.Slug}
	} else {
		// create a new post with this slug
		_, err := posts.InsertOne(ctx, post)
		return err
	}
}

// given a Post struct, update the post with a matching slug in the DB
// or return an error if the UpdateOne fails
func (pm *PostModel) Update(ctx context.Context, post Post) error {
	posts := pm.client.Database(pm.dbName).Collection("posts")

	ur, err := posts.UpdateOne(ctx, bson.M{"slug": post.Slug}, bson.M{"$set": post})
	if err != nil {
		return err
	} else if ur.ModifiedCount == 0 {
		return errors.New("post update failed")
	}
	return nil
}

// given a username, return a list of posts that belong to that username
func (pm *PostModel) GetByUsername(ctx context.Context, username string) ([]Post, error) {
	posts := pm.client.Database(pm.dbName).Collection("posts")

	var postSlice []Post
	cur, err := posts.Find(ctx, bson.M{"owner_username": username})
	if err != nil {
		return []Post{}, err
	}

	err = cur.All(ctx, &postSlice)
	if err != nil {
		return []Post{}, err
	}

	return postSlice, nil
}

type Post struct {
	OwnerUsername string `bson:"owner_username"`
	Title         string
	Subtitle      string
	Author        string
	Content       string
	Slug          string
	Pubdate       time.Time
}

func NewPost(username, title, subtitle, author, content, slug string, pubdate time.Time) *Post {
	return &Post{
		OwnerUsername: username,
		Title:         title,
		Subtitle:      subtitle,
		Author:        author,
		Content:       content,
		Slug:          slug,
		Pubdate:       pubdate,
	}
}
