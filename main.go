package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Comment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Text        string             `bson:"blog_text"`
	CommentDate primitive.DateTime `bson:"comment_date"`
	UpVote      int                `bson:"up_votes"`
	DownVote    int                `bson:"down_votes"`
}

// models
type Blog struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty"`
	Content       string               `bson:"content,omitempty"`
	Comments      []primitive.ObjectID `bson:"comments"`
	PublishedDate primitive.DateTime   `bson:"pub_date"`
}

// mongo configuration
func initDb() *mongo.Client {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	return client
}

var client = initDb()

var usersCollection = client.Database("blogdb").Collection("users")
var blogCollection = client.Database("blogdb").Collection("blogs")
var commentCollection = client.Database("blogdb").Collection("comments")

func Ping(c *gin.Context) {
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
	c.String(200, "ping")
}

// user specific handlers
func Signup(c *gin.Context) {
	c.String(200, "Signup")
}
func Login(c *gin.Context) {
	c.String(200, "login")
}
func Logout(c *gin.Context) {
	c.String(200, "logout")
}

// blog specific handlers

// get all blogs
func GetAllBlogs(c *gin.Context) {
	// getting values out
	cursor, err := blogCollection.Find(context.TODO(), bson.D{}) // no filter
	if err != nil {
		panic(err)
	}
	var blogs []Blog
	if err = cursor.All(context.TODO(), &blogs); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, blogs)
}

// insert new blog
func InsertBlog(c *gin.Context) {
	var pub primitive.DateTime
	blog := Blog{
		Content:       "sample text",
		Comments:      []primitive.ObjectID{},
		PublishedDate: pub,
	}
	resp, err := blogCollection.InsertOne(context.TODO(), blog)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Blog Inserted"})
	}
	c.IndentedJSON(http.StatusOK, resp)
}

func DeleteBlogByID(c *gin.Context) {
	_id := c.Param("blog_id")
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
	}
	deleteFilter := bson.D{{"_id", blog_id}}
	result, err := blogCollection.DeleteOne(context.TODO(), deleteFilter)
	// check for errors in the deleting
	if err != nil {
		panic(err)
	}
	// display the number of documents deleted
	type replyJson struct {
		DeletedCount int
	}
	reply := replyJson{
		DeletedCount: int(result.DeletedCount),
	}
	c.IndentedJSON(http.StatusOK, reply)
}

// insert comments by id
func InsertCommentsByBlogID(c *gin.Context) {
	_id := c.Param("blog_id")
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
	}

	var cdate primitive.DateTime

	comment := Comment{
		Text:        "This is sample comment",
		CommentDate: cdate,
		UpVote:      0,
		DownVote:    0,
	}
	comment_id, err := commentCollection.InsertOne(context.TODO(), comment)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt insert comment"})
	}

	searchFilter := bson.D{{"_id", blog_id}}
	var result Blog
	// check for errors in the finding
	if err = blogCollection.FindOne(context.TODO(), searchFilter).Decode(&result); err != nil {
		panic(err)
	}

	comments := result.Comments
	newComment, ok := comment_id.InsertedID.(primitive.ObjectID)
	if ok {
		comments = append(comments, newComment)
	}

	// update query
	update := bson.D{
		{"$set",
			bson.D{
				{"comments", comments},
			},
		},
	}
	// fetch blog by id and update comment into it
	resp, err := blogCollection.UpdateByID(context.TODO(), blog_id, update)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt Find"})
	}
	c.IndentedJSON(http.StatusOK, resp.ModifiedCount)
}

func DeleteComments(c *gin.Context) {

}

func main() {
	r := gin.Default()
	// attaching handlers

	// mongo check
	r.GET("/ping", Ping)
	// blogs
	r.GET("/blogs", GetAllBlogs)
	r.POST("/blog/insert", InsertBlog)
	r.DELETE("/blog/:id", DeleteBlogByID)
	// comments
	r.POST("/comments/:blog_id", InsertCommentsByBlogID)
	r.DELETE("/comments/:blog_id/comment_id", DeleteComments)
	r.Run()
}
