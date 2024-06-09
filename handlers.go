package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func Ping(c *gin.Context) {
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
	c.String(200, "ping")
}

func authenticateUser(c *gin.Context) {
	token, err := c.Cookie("token")
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}
	if VerifyToken(token) != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}
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
	md5Password := fmt.Sprintf("%X", md5.Sum([]byte(req.Password)))
	user := User{Name: req.Username, Password: md5Password, Description: req.Description}
	_, err := UsersCollection.InsertOne(context.TODO(), user)
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
	md5Password := fmt.Sprintf("%X", md5.Sum([]byte(req.Password)))
	// fetching info
	searchFilter := bson.D{{"name", req.Username}}
	var user User
	if err := UsersCollection.FindOne(context.TODO(), searchFilter).Decode(&user); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"Error": "Invalid username or password"})
		return
	}

	if user.Password != md5Password {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"Error": "Invalid username or password"})
	}
	tokenString, err := CreateToken(req.Username)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"Error": "Invalid username or password"})
		return
	}
	c.SetCookie("token", tokenString, 3600, "/", "localhost", false, false)
	c.IndentedJSON(http.StatusOK, gin.H{"Message": "Login successful"})
}
func Logout(c *gin.Context) {
	c.String(200, "logout")
}

func GetAllUsers(c *gin.Context) {
	authenticateUser(c)

	cursor, err := UsersCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"Error": "Something bad happened, please try again"})
		return
	}
	var users []User
	if err = cursor.All(context.TODO(), &users); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"Error": "Something bad happened, please try again"})
		return
	}
	c.IndentedJSON(http.StatusOK, users)
}
func GetUserByID(c *gin.Context) {
	authenticateUser(c)

	_id := c.Param("id")
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
	}
	searchFilter := bson.D{{"_id", userId}}
	var user User
	if err = UsersCollection.FindOne(context.TODO(), searchFilter).Decode(&user); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, user)
}
func DeleteUserByID(c *gin.Context) {
	authenticateUser(c)

	_id := c.Param("id")
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
		return
	}
	deleteFilter := bson.D{{"_id", userId}}
	result, err := UsersCollection.DeleteOne(context.TODO(), deleteFilter)
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
	authenticateUser(c)

	cursor, err := BlogCollection.Find(context.TODO(), bson.D{})
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
	authenticateUser(c)
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
	resp, err := BlogCollection.InsertOne(context.TODO(), blog)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Blog Inserted"})
	}
	c.IndentedJSON(http.StatusOK, resp)
}

func DeleteBlogByID(c *gin.Context) {
	authenticateUser(c)
	_id := c.Param("id")
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
		return
	}
	deleteFilter := bson.D{{"_id", blog_id}}
	result, err := BlogCollection.DeleteOne(context.TODO(), deleteFilter)
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
	authenticateUser(c)
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
	comment_id, err := CommentCollection.InsertOne(context.TODO(), comment)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt insert comment"})
		return
	}

	searchFilter := bson.D{{"_id", blog_id}}
	var result Blog
	// check for errors in the finding
	if err = BlogCollection.FindOne(context.TODO(), searchFilter).Decode(&result); err != nil {
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
	resp, err := BlogCollection.UpdateByID(context.TODO(), blog_id, update)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt Find"})
		return
	}
	c.IndentedJSON(http.StatusOK, resp.ModifiedCount)
}

func DeleteComments(c *gin.Context) {
	authenticateUser(c)

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
	if err = BlogCollection.FindOne(context.TODO(), searchFilter).Decode(&result); err != nil {
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
	_, err = BlogCollection.UpdateByID(context.TODO(), blogId, update)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt Find"})
		return
	}

	deleteFilter := bson.D{{"_id", commentId}}
	deleteResult, err := CommentCollection.DeleteOne(context.TODO(), deleteFilter)
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
	authenticateUser(c)

	cursor, err := CommentCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var comments []Comment
	if err = cursor.All(context.TODO(), &comments); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, comments)
}
