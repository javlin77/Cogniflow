package routes

import (
	"authentication/controllers"
	"authentication/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.RouterGroup) {
	router.POST("/signup", controllers.Signup())
	router.POST("/login", controllers.Login())
	router.POST("/forgot-password", controllers.ForgotPassword())
	router.POST("/reset-password", controllers.ResetPassword())
	protected := router.Group("/")
	protected.Use(middleware.Authenticate())
	{
		// Current user (all authenticated)
		protected.GET("/me", controllers.GetMe())

		// ADMIN only
		protected.GET("/users",
			middleware.Authorize("ADMIN"),
			controllers.GetUsers(),
		)
		protected.GET("/admin/high-risk",
			middleware.Authorize("ADMIN"),
			controllers.GetHighRiskUsers(),
		)

		// USER (self) + ADMIN
		protected.GET("/user/:id",
			middleware.Authorize("ADMIN", "USER"),
			controllers.GetUser(),
		)

		// Fatigue / study sessions (authenticated users)
		protected.POST("/study-sessions", controllers.CreateStudySession())
		protected.GET("/study-sessions", controllers.GetMySessions())
		protected.GET("/fatigue-scores", controllers.GetMyFatigueScores())
	}
}
