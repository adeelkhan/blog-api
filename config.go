package main

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var SecretKey = []byte("secret-key")

func initDb() *mongo.Client {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}
	return client
}

var client = initDb()

var UsersCollection = client.Database("blogdb").Collection("users")
var BlogCollection = client.Database("blogdb").Collection("blogs")
var CommentCollection = client.Database("blogdb").Collection("comments")
