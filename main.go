package main

import (
	"github.com/gin-gonic/gin"
)

// mongo configuration

func main() {
	r := gin.Default()

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
