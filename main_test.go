package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/assert"
)

func SetUpRouter() *gin.Engine {
	router := gin.Default()
	return router
}

var testDb *TestDatabase

func TestMain(m *testing.M) {
	testDb = SetupTestDatabase()
	// override to testdb
	db = testDb.DbInstance
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	testDb.TearDown()
}

func TestRegistration(t *testing.T) {
	r := SetUpRouter()
	r.POST("/users/register", Register)
	registrationRequest := RegisterRequest{
		Username:    "testuser",
		Password:    "testpass",
		Description: "test-description",
	}
	jsonValue, _ := json.Marshal(registrationRequest)
	req, _ := http.NewRequest("POST", "/users/register", bytes.NewBuffer(jsonValue))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	fmt.Println(w)
	assert.Equal(t, http.StatusCreated, w.Code)
}
