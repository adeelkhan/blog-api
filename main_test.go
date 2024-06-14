package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gotest.tools/assert"
)

func SetUpRouter() *gin.Engine {
	r := gin.Default()
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
	return r
}

var testDb *TestDatabase
var router *gin.Engine
var authTokenString string
var testUser = map[string]string{
	"username":    "testuser",
	"password":    "testpassword",
	"description": "test-description",
}

func SetUp() {
	testDb = SetupTestDatabase()
	router = SetUpRouter()
	// override to testdb
	db = testDb.DbInstance
	SetUpMockData(db)
	authTokenString, _ = CreateToken(testUser["username"])
}

func TestMain(m *testing.M) {
	SetUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	testDb.TearDown()
}

// helper functions
func SetUpMockData(db *mongo.Database) {
	// registering user
	md5Password := fmt.Sprintf("%X", md5.Sum([]byte(testUser["password"])))
	user := User{Name: testUser["username"], Password: md5Password, Description: testUser["description"]}
	resp, _ := db.Collection("users").InsertOne(context.TODO(), user)
	userId := resp.InsertedID.(primitive.ObjectID)
	testUser["ID"] = resp.InsertedID.(primitive.ObjectID).Hex()

	// inserting blog data
	br := Blog{
		Content: "test-blog",
	}
	resp, _ = db.Collection("blogs").InsertOne(context.TODO(), br)
	blogId := resp.InsertedID.(primitive.ObjectID)
	testUser["blogID"] = blogId.Hex()

	brecord := BlogRecord{
		UserID: userId,
		BlogID: blogId,
	}

	_, _ = db.Collection("blogrecords").InsertOne(context.TODO(), brecord)

	// inserting comments
	comment := Comment{
		Text:        "test-comments",
		CommentDate: time.Now(),
		UpVote:      0,
		DownVote:    0,
	}
	comment_id, _ := db.Collection("comments").InsertOne(context.TODO(), comment)

	searchFilter := bson.D{{"_id", blogId}}
	var result Blog
	// check for errors in the finding
	_ = db.Collection("blogs").FindOne(context.TODO(), searchFilter).Decode(&result)

	comments := result.Comments
	newComment, ok := comment_id.InsertedID.(primitive.ObjectID)
	testUser["commentID"] = newComment.Hex()
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
	_, _ = db.Collection("blogs").UpdateByID(context.TODO(), blogId, update)
}

func TestRegistrationCreated(t *testing.T) {
	registrationRequest := RegisterRequest{
		Username:    "test-username",
		Password:    "test-password",
		Description: "test-description",
	}
	jsonValue, _ := json.Marshal(registrationRequest)
	req, _ := http.NewRequest("POST", "/users/register", bytes.NewBuffer(jsonValue))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLoginSuccess(t *testing.T) {
	loginRequest := LoginRequest{
		Username: testUser["username"],
		Password: testUser["password"],
	}
	jsonValue, _ := json.Marshal(loginRequest)
	req, _ := http.NewRequest("POST", "/users/login", bytes.NewBuffer(jsonValue))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAllUsers(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}

	cookies := []string{cookieToken.String()}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users", nil)
	req.Header["Cookie"] = cookies

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserByID(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}
	req, _ := http.NewRequest("GET", fmt.Sprintf("/users/%s", testUser["ID"]), nil)
	req.Header["Cookie"] = []string{cookieToken.String()}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInsertBlog(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}
	br := BlogRequest{
		Content: "test content",
	}
	jsonValue, _ := json.Marshal(br)
	req, _ := http.NewRequest("POST", "/blog/insert", bytes.NewBuffer(jsonValue))
	req.Header["Cookie"] = []string{cookieToken.String()}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	fmt.Println(w)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInsertCommentByBlogID(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}
	cr := CommentRequest{
		Comment: "test-comment",
	}
	jsonValue, _ := json.Marshal(cr)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/comments/insert/%s", testUser["blogID"]), bytes.NewBuffer(jsonValue))
	req.Header["Cookie"] = []string{cookieToken.String()}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAllComments(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}
	req, _ := http.NewRequest("GET", "/comments/", nil)
	req.Header["Cookie"] = []string{cookieToken.String()}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteComments(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}
	path := fmt.Sprintf("/comments/delete/%s/%s", testUser["blogID"], testUser["commentID"])
	req, _ := http.NewRequest("DELETE", path, nil)
	req.Header["Cookie"] = []string{cookieToken.String()}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteBlogByID(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/blog/%s", testUser["blogID"]), nil)
	req.Header["Cookie"] = []string{cookieToken.String()}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteUserByID(t *testing.T) {
	cookieToken := &http.Cookie{
		Name:     "token",
		Value:    authTokenString,
		HttpOnly: false,
		MaxAge:   3600,
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
	}
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/users/%s", testUser["ID"]), nil)
	req.Header["Cookie"] = []string{cookieToken.String()}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
