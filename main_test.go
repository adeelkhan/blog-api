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

	"github.com/gin-gonic/gin"
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
	_, _ = db.Collection("users").InsertOne(context.TODO(), user)

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
