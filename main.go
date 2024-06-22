package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/adeelkhan/blog-app/proto/blogpb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	// this must be embedded
	blogpb.UnimplementedBlogServiceServer
}

func (*server) Login(ctx context.Context, req *blogpb.LoginRequest) (*blogpb.LoginResponse, error) {
	md5Password := fmt.Sprintf("%X", md5.Sum([]byte(req.GetPassword())))
	// fetching info
	searchFilter := bson.D{{"name", req.GetName()}}
	var user User
	if err := db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&user); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid username and password ;%v", err),
		)
	}

	if user.Password != md5Password {
		return nil, status.Errorf(
			codes.Internal,
			"Invalid username and password",
		)
	}
	tokenString, err := CreateToken(req.GetName())
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid username and password ;%v", err),
		)
	}
	response := &blogpb.LoginResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusCreated,
			Message: "Login Successful",
		},
		Token: tokenString,
	}
	return response, nil
}

// func (*server) LogOut(ctx context.Context, req *blogpb.LogoutRequest) (*blogpb.LogoutResponse, error) {
// }
func (*server) Register(ctx context.Context, req *blogpb.RegisterRequest) (*blogpb.RegisterResponse, error) {
	searchFilter := bson.D{{"name", req.GetName()}}
	var user User
	if err := db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&user); err != mongo.ErrNoDocuments {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}
	if user.Name != "" {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Username already exist, choose a different name."),
		)
	}

	md5Password := fmt.Sprintf("%X", md5.Sum([]byte(req.GetPassword())))
	user = User{Name: req.GetName(), Password: md5Password, Description: req.GetDescription()}
	_, err := db.Collection("users").InsertOne(context.TODO(), user)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("User not Registered ;%v", err),
		)
	}
	response := &blogpb.RegisterResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusCreated,
			Message: "User Registered Successfully",
		},
	}
	return response, nil
}

func (*server) GetAllUsers(ctx context.Context, req *blogpb.UserRequest) (*blogpb.UserResponse, error) {
	token := req.GetToken()
	_, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}
	cursor, err := db.Collection("users").Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid username and password ;%v", err),
		)
	}

	var users []User
	if err = cursor.All(context.TODO(), &users); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Something bad happened, please try again ;%v", err),
		)
	}
	busers := make([]*blogpb.BlogUser, 0)
	for i := range users {
		user := users[i]
		b := &blogpb.BlogUser{
			Id:          user.ID.Hex(),
			Name:        user.Name,
			Description: user.Description,
		}
		busers = append(busers, b)
	}

	response := &blogpb.UserResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "User retreive successfull",
		},
		User: busers,
	}
	return response, nil
}

func (*server) GetUsersByID(ctx context.Context, req *blogpb.UserByIDRequest) (*blogpb.UserByIDResponse, error) {
	token := req.GetToken()
	_, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}

	_id := req.GetId()
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid id ;%v", err),
		)
	}
	searchFilter := bson.D{{"_id", userId}}
	var user User
	if err = db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&user); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}

	response := &blogpb.UserByIDResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "User retreive successfull",
		},
		User: &blogpb.BlogUser{
			Id:          user.ID.Hex(),
			Name:        user.Name,
			Description: user.Description,
		},
	}
	return response, nil
}

func (*server) DeleteUserByID(ctx context.Context, req *blogpb.DeleteUserByIDRequest) (*blogpb.DeleteUserByIDResponse, error) {
	token := req.GetToken()
	_, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}

	_id := req.GetId()
	userId, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid id ;%v", err),
		)
	}
	deleteFilter := bson.D{{"_id", userId}}
	_, err = db.Collection("users").DeleteOne(context.TODO(), deleteFilter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}

	response := &blogpb.DeleteUserByIDResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "User delete successfull",
		},
	}
	return response, nil
}

func (*server) GetAllBlogs(ctx context.Context, req *blogpb.AllBlogsRequest) (*blogpb.AllBlogsResponse, error) {
	token := req.GetToken()
	user, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
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
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}
	var blogsRec []BlogRecord
	if err = cursor.All(context.TODO(), &blogsRec); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}

	arr := make([]primitive.ObjectID, 0)
	for i := range blogsRec {
		b := blogsRec[i]
		arr = append(arr, b.BlogID)
	}

	blogFilter := bson.M{"_id": bson.M{"$in": arr}}
	cursor, err = db.Collection("blogs").Find(context.TODO(), blogFilter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}
	var blogs []Blog
	if err = cursor.All(context.TODO(), &blogs); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}

	blogResp := make([]*blogpb.Blog, 0)
	for i := range blogs {
		blog := blogs[i]
		b := &blogpb.Blog{
			Id:      blog.ID.Hex(),
			Content: blog.Content,
		}
		blogResp = append(blogResp, b)
	}

	response := &blogpb.AllBlogsResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "Blogs retrieved successful.",
		},
		Blogs: blogResp,
	}
	return response, nil

}
func (*server) InsertBlog(ctx context.Context, req *blogpb.InsertBlogRequest) (*blogpb.InsertBlogResponse, error) {
	token := req.GetToken()
	user, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}

	blog := Blog{
		Content:       req.Content,
		Comments:      []primitive.ObjectID{},
		PublishedDate: time.Now(),
	}
	respBlog, err := db.Collection("blogs").InsertOne(context.TODO(), blog)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Some error occured while inserting blog ;%v", err),
		)
	}

	blogId := respBlog.InsertedID.(primitive.ObjectID)

	searchFilter := bson.D{{"name", user}}
	var userResponse User
	if err = db.Collection("users").FindOne(context.TODO(), searchFilter).Decode(&userResponse); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}

	// creating blog record
	brecord := BlogRecord{
		UserID: userResponse.ID,
		BlogID: blogId,
	}

	_, err = db.Collection("blogrecords").InsertOne(context.TODO(), brecord)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Some error occurred while inserting blog record ;%v", err),
		)
	}

	response := &blogpb.InsertBlogResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "Blog inserted successful",
		},
		BlogId: blogId.Hex(),
	}
	return response, nil
}
func (*server) DeleteBlogByID(ctx context.Context, req *blogpb.DeleteBlogByIDRequest) (*blogpb.DeleteBlogByIDResponse, error) {
	token := req.GetToken()
	_, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}

	_id := req.GetId()
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid id;%v", err),
		)
	}
	deleteFilter := bson.D{{"_id", blog_id}}
	_, err = db.Collection("blogs").DeleteOne(context.TODO(), deleteFilter)
	// check for errors in the deleting
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
	}

	response := &blogpb.DeleteBlogByIDResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "Blog delete successfull",
		},
	}
	return response, nil
}
func (*server) InsertCommentsByBlogID(ctx context.Context, req *blogpb.InsertCommentByBlogIDRequest) (*blogpb.InsertCommentByBlogIDResponse, error) {
	token := req.GetToken()
	_, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}

	_id := req.GetBlogId()
	blog_id, err := primitive.ObjectIDFromHex(_id)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid id ;%v", err),
		)
	}

	// var cdate time.Time

	comment := Comment{
		Text:        req.GetComment(),
		CommentDate: time.Now(),
		UpVote:      0,
		DownVote:    0,
	}
	comment_id, err := db.Collection("comments").InsertOne(context.TODO(), comment)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Could'nt insert comment ;%v", err),
		)
	}

	searchFilter := bson.D{{"_id", blog_id}}
	var result Blog
	// check for errors in the finding
	if err = db.Collection("blogs").FindOne(context.TODO(), searchFilter).Decode(&result); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error ;%v", err),
		)
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
	_, err = db.Collection("blogs").UpdateByID(context.TODO(), blog_id, update)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal ;%v", err),
		)
	}
	response := &blogpb.InsertCommentByBlogIDResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "Comments inserted successfull",
		},
	}
	return response, nil
}

func (*server) DeleteComments(ctx context.Context, req *blogpb.DeleteCommentByBlogIDRequest) (*blogpb.DeleteCommentByBlogIDResponse, error) {
	token := req.GetToken()
	_, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}

	Id := req.GetBlogId()
	cId := req.GetCommentId()

	blogId, err := primitive.ObjectIDFromHex(Id)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid blog id ;%v", err),
		)
	}
	commentId, err := primitive.ObjectIDFromHex(cId)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Invalid comment id ;%v", err),
		)
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
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal ;%v", err),
		)
	}

	deleteFilter := bson.D{{"_id", commentId}}
	_, err = db.Collection("comments").DeleteOne(context.TODO(), deleteFilter)
	// check for errors in the deleting
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal ;%v", err),
		)
	}
	response := &blogpb.DeleteCommentByBlogIDResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "Comments deleted successfull",
		},
	}
	return response, nil
}

func (*server) DeleteAllComments(ctx context.Context, req *blogpb.DeleteAllCommentsRequest) (*blogpb.DeleteAllCommentsResponse, error) {
	token := req.GetToken()
	_, err := authenticateUser(token)
	if err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			fmt.Sprintf("Permission denied ;%v", err),
		)
	}

	cursor, err := db.Collection("comments").Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var comments []Comment
	if err = cursor.All(context.TODO(), &comments); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal ;%v", err),
		)
	}

	response := &blogpb.DeleteAllCommentsResponse{
		RpcResp: &blogpb.ResponseStatus{
			Status:  http.StatusOK,
			Message: "All Comments deleted successfully",
		},
	}
	return response, nil
}

func main() {
	// default for grpc
	fmt.Println("Starting Blog Service")
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)
	blogpb.RegisterBlogServiceServer(s, &server{})

	// registering reflection service on gRPC server.
	reflection.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	fmt.Println("Blog Service Started")

	// wait for Control c to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	fmt.Println("Stopping the server")
	s.Stop()
	fmt.Println("Closing the listener")
	lis.Close()
}
