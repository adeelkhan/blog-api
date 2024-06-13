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
	"go.mongodb.org/mongo-driver/mongo"
)

type RegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type replyJson struct {
	DeletedCount int
}
type CommentRequest struct {
	Comment string `json:"comment" binding:"required"`
}

func authenticateUser(c *gin.Context) (error, string) {
	token, err := c.Cookie("token")
	if err != nil {
		return err, ""
	}
	var user string
	if err, user = VerifyToken(token); err != nil {
		return err, ""
	}
	return nil, user
}

// user specific handlers
func Register(c *gin.Context) {
	req := RegisterRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	searchFilter := bson.D{{"name", req.Username}}
	var user User
	if err := db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&user); err != mongo.ErrNoDocuments {
		fmt.Println(err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"Error": "Invalid username or password"})
		return
	}
	if user.Name != "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"Error": "Username already exist, choose a different name."})
		return
	}

	md5Password := fmt.Sprintf("%X", md5.Sum([]byte(req.Password)))
	user = User{Name: req.Username, Password: md5Password, Description: req.Description}
	_, err := db.Collection("users").InsertOne(context.TODO(), user)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"Error": "User not Registered"})
	}
	c.IndentedJSON(http.StatusCreated, gin.H{"Message": "User Registered successful"})
}

func Login(c *gin.Context) {

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
	if err := db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&user); err != nil {
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
	err, _ := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

	cursor, err := db.Collection("users").Find(context.TODO(), bson.D{})
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
	err, _ := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

	_id := c.Param("id")
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
	}
	searchFilter := bson.D{{"_id", userId}}
	var user User
	if err = db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&user); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, user)
}
func DeleteUserByID(c *gin.Context) {
	err, _ := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

	_id := c.Param("id")
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
		return
	}
	deleteFilter := bson.D{{"_id", userId}}
	result, err := db.Collection("users").DeleteOne(context.TODO(), deleteFilter)
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
	err, user := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}
	// find user id
	searchFilter := bson.D{{"name", user}}
	var userResponse User
	if err = db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&userResponse); err != nil {
		panic(err)
	}

	// find user blogs records
	filter := bson.D{{"user_id", userResponse.ID}}
	cursor, err := db.Collection("blogrecords").Find(context.TODO(), filter)
	if err != nil {
		panic(err)
	}
	var blogsRec []BlogRecord
	if err = cursor.All(context.TODO(), &blogsRec); err != nil {
		panic(err)
	}
	arr := make([]primitive.ObjectID, 0)
	for i := range blogsRec {
		b := blogsRec[i]
		arr = append(arr, b.BlogID)
	}

	blogFilter := bson.M{"_id": bson.M{"$in": arr}}
	cursor, err = db.Collection("blogrecords").Find(context.TODO(), blogFilter)
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
	err, user := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

	type BlogRequest struct {
		Content string `json:"content" binding:"required"`
	}
	req := BlogRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	blog := Blog{
		Content:       req.Content,
		Comments:      []primitive.ObjectID{},
		PublishedDate: time.Now(),
	}
	respBlog, err := db.Collection("blogs").InsertOne(context.TODO(), blog)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Some error occurred while inserting blog"})
		return
	}
	blogId := respBlog.InsertedID.(primitive.ObjectID)

	searchFilter := bson.D{{"name", user}}
	var userResponse User
	if err = db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&userResponse); err != nil {
		panic(err)
	}

	// creating blog record
	brecord := BlogRecord{
		UserID: userResponse.ID,
		BlogID: blogId,
	}

	_, err = db.Collection("blogrecords").InsertOne(context.TODO(), brecord)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Some error occurred while inserting blog record"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": "Blog inserted successful"})
}

func DeleteBlogByID(c *gin.Context) {
	err, _ := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

	_id := c.Param("id")
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
		return
	}
	deleteFilter := bson.D{{"_id", blog_id}}
	result, err := db.Collection("blogs").DeleteOne(context.TODO(), deleteFilter)
	// check for errors in the deleting
	if err != nil {
		panic(err)
	}
	// display the number of documents deleted
	reply := replyJson{
		DeletedCount: int(result.DeletedCount),
	}
	c.IndentedJSON(http.StatusOK, reply)
}

func InsertCommentsByBlogID(c *gin.Context) {
	err, _ := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

	_id := c.Param("blog_id")
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id"})
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
	comment_id, err := db.Collection("comments").InsertOne(context.TODO(), comment)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt insert comment"})
		return
	}

	searchFilter := bson.D{{"_id", blog_id}}
	var result Blog
	// check for errors in the finding
	if err = db.Collection("blogs").FindOne(context.TODO(), searchFilter).Decode(&result); err != nil {
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
	resp, err := db.Collection("blogs").UpdateByID(context.TODO(), blog_id, update)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt Find"})
		return
	}
	c.IndentedJSON(http.StatusOK, resp.ModifiedCount)
}

func DeleteComments(c *gin.Context) {
	err, _ := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

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
	if err = db.Collection("blogs").FindOne(context.TODO(), searchFilter).Decode(&result); err != nil {
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
	_, err = db.Collection("blogs").UpdateByID(context.TODO(), blogId, update)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldnt Find"})
		return
	}

	deleteFilter := bson.D{{"_id", commentId}}
	deleteResult, err := db.Collection("comments").DeleteOne(context.TODO(), deleteFilter)
	// check for errors in the deleting
	if err != nil {
		panic(err)
	}
	// display the number of documents deleted
	reply := replyJson{
		DeletedCount: int(deleteResult.DeletedCount),
	}
	c.IndentedJSON(http.StatusOK, reply)
}

func GetAllComments(c *gin.Context) {
	err, _ := authenticateUser(c)
	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"Error": "Invalid token"})
		return
	}

	cursor, err := db.Collection("comments").Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var comments []Comment
	if err = cursor.All(context.TODO(), &comments); err != nil {
		panic(err)
	}
	c.IndentedJSON(http.StatusOK, comments)
}
