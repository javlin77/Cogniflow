package main

import (
	"authentication/config"
	"authentication/helpers"
	"authentication/routes"
	"fmt"

	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	//Connect to mongoDB

	log.Println("Starting application...")

	key := config.GenerateRandomKey()
	helpers.SetJWTKey(key)
	fmt.Printf("Generated Key: %s\n", key)
	//Init gin router
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "running",
		})
	})

	api := r.Group("/api")
	routes.SetupRoutes(api)

	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) { c.File("./static/index.html") })
	r.GET("/login", func(c *gin.Context) { c.File("./static/index.html") })
	r.GET("/signup", func(c *gin.Context) { c.File("./static/signup.html") })
	r.GET("/forgot-password", func(c *gin.Context) { c.File("./static/forgot-password.html") })
	r.GET("/reset-password", func(c *gin.Context) { c.File("./static/reset-password.html") })
	r.GET("/dashboard", func(c *gin.Context) { c.File("./static/dashboard.html") })

	// //Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
	log.Println("Server is running on http://localhost:" + port)
}
