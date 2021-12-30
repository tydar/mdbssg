package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/google/uuid"
)

// UserModel implements an interface for access to User data
type UserModel struct {
	client *mongo.Client
	dbName string
}

func NewUserModel(client *mongo.Client, dbName string) *UserModel {
	return &UserModel{
		client: client,
		dbName: dbName,
	}
}

// User is the model for documents in the users collection in the db
type User struct {
	DisplayName    string `bson:"display_name,omitempty"`
	Username       string
	PasswordHashed []byte    `bson:"password"`
	CreatedAt      time.Time `bson:"created_at"`
}

type Session struct {
	Token     string
	ExpiresAt time.Time `bson:"expires_at"`
}

type Sessions struct {
	Sessions []Session
}

func (u *UserModel) getSessionsByUsername(ctx context.Context, username string) ([]Session, error) {
	var sessions Sessions
	opts := options.FindOne().SetProjection(bson.M{"sessions": 1})
	users := u.client.Database(u.dbName).Collection("users")
	sr := users.FindOne(ctx, bson.M{"username": username}, opts)

	if sr.Err() != nil {
		return []Session{}, sr.Err()
	}

	sr.Decode(&sessions)
	return sessions.Sessions, nil
}

// check if a session is valid, and if it is expired, slice it out
func (u *UserModel) CheckSessionValid(ctx context.Context, username, token string) bool {
	sessions, err := u.getSessionsByUsername(ctx, username)
	if err == mongo.ErrNoDocuments {
		fmt.Println("no sessions for this user")
		return false
	}

	sessIdx := -1
	for i := range sessions {
		if sessions[i].Token == token {
			sessIdx = i
		}
	}

	if sessIdx == -1 {
		fmt.Println("session not found")
		fmt.Printf("%s\n%+v\n", token, sessions)
		return false
	}

	users := u.client.Database(u.dbName).Collection("users")

	newSess := make([]Session, 0)
	if sessIdx < len(sessions)-1 {
		newSess = append(sessions[0:sessIdx], sessions[sessIdx+1:]...)
	} else {
		newSess = sessions[0:sessIdx]
	}

	if sessions[sessIdx].ExpiresAt.Before(time.Now()) {
		_, err = users.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"sessions": newSess}})
		if err != nil {
			// TODO: determine the best way to handle this error
			fmt.Println(err)
		}
		fmt.Println("session expired")
		return false
	}

	// for now, if the session has been checked (implying the session is active), set its ExpiresAt to 5 min
	// in the future
	_, err = users.UpdateOne(ctx,
		bson.M{"username": username, "sessions.token": sessions[sessIdx].Token},
		bson.M{"$set": bson.M{"sessions.$.expires_at": time.Now().Add(5 * time.Minute)}})

	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (u *UserModel) AppendNewSession(ctx context.Context, username string) (string, error) {
	var sessions []Session
	sessions, err := u.getSessionsByUsername(ctx, username)
	if err == mongo.ErrNoDocuments {
		sessions = make([]Session, 0)
	}

	session := Session{
		Token:     uuid.NewString(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	sessions = append(sessions, session)

	users := u.client.Database(u.dbName).Collection("users")
	_, err = users.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"sessions": sessions}})
	if err != nil {
		return "", err
	}

	_, err = u.removeExpiredSessions(ctx, username)
	if err != nil {
		fmt.Println("no expired sessions removed")
	}

	return session.Token, nil
}

// given a username and a token, invalidate that session by deleting it from the DB
func (u *UserModel) InvalidateSession(ctx context.Context, username, token string) error {
	sessions, err := u.getSessionsByUsername(ctx, username)
	if err != nil {
		return err
	}

	sessIdx := -1
	for i := range sessions {
		if sessions[i].Token == token {
			sessIdx = i
		}
	}

	if sessIdx == -1 {
		return errors.New("session not found")
	}

	users := u.client.Database(u.dbName).Collection("users")

	newSess := make([]Session, 0)
	if sessIdx < len(sessions)-1 {
		newSess = append(sessions[0:sessIdx], sessions[sessIdx+1:]...)
	} else {
		newSess = sessions[0:sessIdx]
	}

	_, err = users.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"sessions": newSess}})
	return err
}

// given a username, remove expired sessions with an updateMany(...{ '$pull': ... })
func (u *UserModel) removeExpiredSessions(ctx context.Context, username string) (int, error) {
	sessions, err := u.getSessionsByUsername(ctx, username)
	if err != nil {
		return 0, err
	}

	sessTokens := make([]string, 0)
	for i := range sessions {
		if sessions[i].ExpiresAt.Before(time.Now()) {
			sessTokens = append(sessTokens, sessions[i].Token)
		}
	}

	users := u.client.Database(u.dbName).Collection("users")

	// note: db.collection.updateMany() is not atomic. I think this could cause issues if a user is trying
	// to sign in at one location (so this method is called) while that account is in use at another location
	mr, err := users.UpdateMany(ctx, bson.M{},
		bson.M{"$pull": bson.M{"sessions": bson.M{"token": bson.M{"$in": sessTokens}}}})
	return int(mr.ModifiedCount), err
}

// validate user info and pass to function to create
// return an error if unable to create user:
// * password hash failed
// * duplicate username
// * password fails rules
// * db store fails
func (u *UserModel) CreateUser(ctx context.Context, username, password, display string) error {
	err := validatePassword(password)
	if err != nil {
		return err
	}

	_, err = u.GetByUsername(ctx, username)
	if err == nil {
		return errors.New("username already in use")
	} else if err != mongo.ErrNoDocuments {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}

	user := User{
		DisplayName:    display,
		Username:       username,
		PasswordHashed: hashed,
		CreatedAt:      time.Now(),
	}
	return u.createUserFromModel(ctx, user)
}

// given a User struct, execute the insert
func (u *UserModel) createUserFromModel(ctx context.Context, user User) error {
	users := u.client.Database(u.dbName).Collection("users")
	_, err := users.InsertOne(ctx, user)
	return err
}

// given a username, return a User if a user with that username exists
func (u *UserModel) GetByUsername(ctx context.Context, username string) (User, error) {
	var user User
	users := u.client.Database(u.dbName).Collection("users")
	opts := options.FindOne().SetSort(bson.M{"created_at": 1})
	err := users.FindOne(ctx, bson.M{"username": username}, opts).Decode(&user)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

// given a User and a password, check if the password matches with bcrypt
func CheckPassword(user User, password string) bool {
	err := bcrypt.CompareHashAndPassword(user.PasswordHashed, []byte(password))
	if err != nil {
		return false
	}
	return true
}

// given a password string, check that it is of appropriate length (>10 bytes <56 bytes for bcrypt)
// error messages don't match reality right now. need a validation step to ensure ASCII only
func validatePassword(password string) error {
	if len([]byte(password)) < 10 {
		return errors.New("password must be greater than 10 characters")
	} else if len([]byte(password)) > 56 {
		return errors.New("password must be less than 56 characters")
	}
	return nil
}

// given a username and new password, validate the password and commit it to the DB
func (u *UserModel) UpdatePassword(ctx context.Context, username, password string) error {
	err := validatePassword(password)
	if err != nil {
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}

	coll := u.client.Database(u.dbName).Collection("users")
	ur, err := coll.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"password": newHash}})
	if err != nil {
		return err
	}

	if ur.ModifiedCount == 0 {
		return errors.New("password update failed: db match error")
	}
	return nil
}
