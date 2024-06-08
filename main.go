package main

import (
	"context"
	"net/http"
	"time"

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
	Name        string             `bson:"name"`
	Description string             `bson:"Description"`
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
func Register(c *gin.Context) {
	type RegisterRequest struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		Description string `json:"description" binding:"required"`
	}
	req := RegisterRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	user := User{Name: req.Username, Description: req.Description}
	_, err := usersCollection.InsertOne(context.TODO(), user)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"Error": "User not Registered"})
	}
	c.IndentedJSON(http.StatusOK, gin.H{"Message": "User Registered successful"})
}
func Login(c *gin.Context) {
	type LoginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	req := LoginRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if req.Username != "username" && req.Password != "password" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"Error": "Invalid username or password"})
	}
	c.IndentedJSON(http.StatusOK, gin.H{"Message": "Login successful"})
}
func Logout(c *gin.Context) {
	c.String(200, "logout")
}

func GetAllUsers(c *gin.Context) {
	// getting values out
	cursor, err := usersCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var users []User
	if err = cursor.All(context.TODO(), &users); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, users)
}
func GetUserByID(c *gin.Context) {
	_id := c.Param("id")
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
	}
	searchFilter := bson.D{{"_id", userId}}
	var user User
	if err = usersCollection.FindOne(context.TODO(), searchFilter).Decode(&user); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, user)
}
func DeleteUserByID(c *gin.Context) {
	_id := c.Param("id")
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
		return
	}
	deleteFilter := bson.D{{"_id", userId}}
	result, err := usersCollection.DeleteOne(context.TODO(), deleteFilter)
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

// blog specific handlers
func GetAllBlogs(c *gin.Context) {
	// getting values out
	cursor, err := blogCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var blogs []Blog
	if err = cursor.All(context.TODO(), &blogs); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, blogs)
}

func InsertBlog(c *gin.Context) {
	type BlogRequest struct {
		Content string `json:"content" binding:"required"`
	}
	req := BlogRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	// var pub time.Time
	blog := Blog{
		Content:       req.Content,
		Comments:      []primitive.ObjectID{},
		PublishedDate: time.Now(),
	}
	resp, err := blogCollection.InsertOne(context.TODO(), blog)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Blog Inserted"})
	}
	c.IndentedJSON(http.StatusOK, resp)
}

func DeleteBlogByID(c *gin.Context) {
	_id := c.Param("id")
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
		return
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

func InsertCommentsByBlogID(c *gin.Context) {
	_id := c.Param("blog_id")
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
	}

	type CommentRequest struct {
		Comment string `json:"comment" binding:"required"`
	}
	req := CommentRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// var cdate time.Time

	comment := Comment{
		Text:        req.Comment,
		CommentDate: time.Now(),
		UpVote:      0,
		DownVote:    0,
	}
	comment_id, err := commentCollection.InsertOne(context.TODO(), comment)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt insert comment"})
		return
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
		return
	}
	c.IndentedJSON(http.StatusOK, resp.ModifiedCount)
}

func DeleteComments(c *gin.Context) {
	Id := c.Param("blog_id")
	cId := c.Param("comment_id")

	blogId, err := primitive.ObjectIDFromHex(Id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid blog id"})
	}
	commentId, err := primitive.ObjectIDFromHex(cId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid comment id"})
	}

	searchFilter := bson.D{{"_id", blogId}}
	var result Blog
	if err = blogCollection.FindOne(context.TODO(), searchFilter).Decode(&result); err != nil {
		panic(err)
	}

	comments := result.Comments
	newComments := make([]primitive.ObjectID, 0)
	for i := range comments {
		if comments[i] != commentId {
			newComments = append(newComments, comments[i])
		}
	}

	// update query
	update := bson.D{
		{"$set",
			bson.D{
				{"comments", newComments},
			},
		},
	}
	// fetch blog by id and update comment into it
	_, err = blogCollection.UpdateByID(context.TODO(), blogId, update)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt Find"})
		return
	}

	deleteFilter := bson.D{{"_id", commentId}}
	deleteResult, err := commentCollection.DeleteOne(context.TODO(), deleteFilter)
	// check for errors in the deleting
	if err != nil {
		panic(err)
	}
	// display the number of documents deleted
	type replyJson struct {
		DeletedCount int
	}
	reply := replyJson{
		DeletedCount: int(deleteResult.DeletedCount),
	}
	c.IndentedJSON(http.StatusOK, reply)
}

func GetAllComments(c *gin.Context) {
	cursor, err := commentCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var comments []Comment
	if err = cursor.All(context.TODO(), &comments); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, comments)
}

func main() {
	r := gin.Default()

	// mongo check
	r.GET("/ping", Ping)

	// users
	r.POST("/users/register", Register)
	r.POST("/users/login", Login)
	r.GET("/users/logout", Logout)

	r.GET("/users", GetAllUsers)
	r.GET("/users/:id", GetUserByID)
	r.DELETE("/users/:id", DeleteUserByID)

	// blogs
	r.GET("/blogs", GetAllBlogs)
	r.POST("/blog/insert", InsertBlog)
	r.DELETE("/blog/:id", DeleteBlogByID)
	// comments
	r.GET("/comments/", GetAllComments)
	r.POST("/comments/insert/:blog_id", InsertCommentsByBlogID)
	r.DELETE("/comments/delete/:blog_id/:comment_id", DeleteComments)
	r.Run()
}
