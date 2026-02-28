package main

import (
	"authentication/config"
	"authentication/helpers"
	"authentication/routes"
	"fmt"

	"log"

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
	r.Run(":8080")
	log.Println("Serever is running on http://localhost:8080")
}
