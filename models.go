package main

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Comment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Text        string             `bson:"blog_text"`
	CommentDate time.Time          `bson:"comment_date"`
	UpVote      int                `bson:"up_votes"`
	DownVote    int                `bson:"down_votes"`
}

// models
type Blog struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty"`
	Content       string               `bson:"content,omitempty"`
	Comments      []primitive.ObjectID `bson:"comments"`
	PublishedDate time.Time            `bson:"pub_date"`
}

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Password    string             `bson:"password"`
	Name        string             `bson:"name"`
	Description string             `bson:"Description"`
}

type BlogRecord struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	UserID primitive.ObjectID `bson:"user_id,omitempty"`
	BlogID primitive.ObjectID `bson:"blog_id,omitempty"`
}
