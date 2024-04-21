package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	docs "websocket/docs"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/joho/godotenv"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func GinMiddleware(allowOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Length, X-CSRF-Token, Token, session, Origin, Host, Connection, Accept-Encoding, Accept-Language, X-Requested-With")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Request.Header.Del("Origin")
		c.Next()
	}
}

func Helloworld(g *gin.Context) {
	g.JSON(http.StatusOK, "helloworld")
}

func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	router := gin.New()
	docs.SwaggerInfo.BasePath = "/"
	v1 := router.Group("/api/v1")
	{
		eg := v1.Group("/example")
		{
			eg.GET("/helloworld", Helloworld)
		}
	}

	router.GET("/ping", Ping)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	server := socketio.NewServer(nil)
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("Socket id connected:", s.ID())
		return nil
	})
	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		log.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})
	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})
	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})
	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("Socket id disconnected", s.ID(), msg)
	})
	go func() {
		if err := server.Serve(); err != nil {
			fmt.Printf("socketio listen error: %s\n", err)
		}
	}()

	defer server.Close()
	router.Use(GinMiddleware("http://127.0.0.1:5500"))
	router.Use(GinMiddleware("http://localhost:3000"))
	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))
	router.StaticFS("/public", http.Dir("./asset"))
	port := os.Getenv("PORT")
	s := fmt.Sprintf("Server listen on:http://localhost:%s", port)
	fmt.Println(s)
	router.Run(":4000")
}
