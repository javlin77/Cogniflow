package controllers

import (
	"authentication/config"
	"authentication/helpers"
	"authentication/models"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var validate = validator.New()
var userCollection = config.OpenCollection("users")

// ===================== SIGNUP =====================
func Signup() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if validationErr := validate.Struct(user); validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Build uniqueness filter: always check email, only check phone when provided.
		orConditions := []bson.M{
			{"email": user.Email},
		}
		if user.Phone != nil {
			if phoneVal := *user.Phone; phoneVal != "" {
				orConditions = append(orConditions, bson.M{"phone": user.Phone})
			}
		}

		count, err := userCollection.CountDocuments(ctx, bson.M{
			"$or": orConditions,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email or phone already exists"})
			return
		}

		// Force default role (SECURE)
		role := "USER"
		user.Role = &role

		user.Password = helpers.HashPassword(user.Password)
		user.Created_at = time.Now()
		user.Updated_at = time.Now()
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()

		accessToken, refreshToken :=
			helpers.GenerateTokens(*user.Email, user.User_id, *user.Role)

		user.Token = &accessToken
		user.Refresh_token = &refreshToken

		_, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": insertErr.Error()})
			return
		}

		// Return token and user so frontend can redirect to dashboard immediately
		user.Password = nil
		user.Token = nil
		user.Refresh_token = nil
		c.JSON(http.StatusOK, gin.H{
			"message":       "User created successfully",
			"token":         accessToken,
			"refresh_token": refreshToken,
			"user":          user,
		})
	}
}

// ===================== LOGIN =====================
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var loginInput models.User
		var foundUser models.User

		// Bind JSON
		if err := c.BindJSON(&loginInput); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		// Safety check (avoid nil pointer panic)
		if loginInput.Email == nil || loginInput.Password == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email and password are required",
			})
			return
		}

		// 🔥 FIXED: Dereference pointer in Mongo query
		err := userCollection.
			FindOne(ctx, bson.M{"email": *loginInput.Email}).
			Decode(&foundUser)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}

		// Verify password
		passwordIsValid, _ :=
			helpers.VerifyPassword(*foundUser.Password, *loginInput.Password)

		if !passwordIsValid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}

		// Generate new tokens
		token, refreshToken :=
			helpers.GenerateTokens(
				*foundUser.Email,
				foundUser.User_id,
				*foundUser.Role,
			)

		helpers.UpdateAllTokens(token, refreshToken, foundUser.User_id)

		// 🔒 Remove sensitive data before sending response
		foundUser.Password = nil
		foundUser.Token = nil
		foundUser.Refresh_token = nil

		c.JSON(http.StatusOK, gin.H{
			"user":          foundUser,
			"token":         token,
			"refresh_token": refreshToken,
		})
	}
}

// ===================== GET CURRENT USER (ME) =====================
func GetMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsValue, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		claims := claimsValue.(*helpers.Claims)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"user_id": claims.UserID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		user.Password = nil
		user.Token = nil
		user.Refresh_token = nil
		user.Reset_token = nil
		user.Reset_expires = nil
		c.JSON(http.StatusOK, user)
	}
}

// ===================== GET SINGLE USER =====================
func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {

		requestedUserId := c.Param("id")

		claimsValue, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		tokenClaims := claimsValue.(*helpers.Claims)

		// USER can only access own data
		if tokenClaims.Role != "ADMIN" &&
			tokenClaims.UserID != requestedUserId {

			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var user models.User

		err := userCollection.
			FindOne(ctx, bson.M{"user_id": requestedUserId}).
			Decode(&user)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		user.Password = nil
		user.Token = nil
		user.Refresh_token = nil

		c.JSON(http.StatusOK, user)
	}
}

// ===================== GET ALL USERS =====================
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cursor, err := userCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var users []models.User
		if err := cursor.All(ctx, &users); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Remove sensitive data
		for i := range users {
			users[i].Password = nil
			users[i].Token = nil
			users[i].Refresh_token = nil
		}

		c.JSON(http.StatusOK, users)
	}
}

// ===================== FORGOT PASSWORD =====================
func ForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var body struct {
			Email *string `json:"email" binding:"required"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
			return
		}

		var foundUser models.User
		err := userCollection.FindOne(ctx, bson.M{"email": *body.Email}).Decode(&foundUser)
		if err != nil {
			// Don't reveal whether email exists
			c.JSON(http.StatusOK, gin.H{
				"message": "If an account exists with this email, you will receive reset instructions.",
			})
			return
		}

		resetToken, err := helpers.GenerateResetToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token"})
			return
		}
		expires := time.Now().Add(1 * time.Hour)
		update := bson.D{
			{"$set", bson.D{
				{"reset_token", resetToken},
				{"reset_expires", expires},
				{"updated_at", time.Now()},
			}},
		}
		_, err = userCollection.UpdateOne(ctx, bson.M{"user_id": foundUser.User_id}, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set reset token"})
			return
		}

		// In dev: return token so frontend can open reset page. In production, send email only.
		c.JSON(http.StatusOK, gin.H{
			"message":     "If an account exists with this email, you will receive reset instructions.",
			"reset_token": resetToken,
		})
	}
}

// ===================== RESET PASSWORD =====================
func ResetPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var body struct {
			Token       string  `json:"token" binding:"required"`
			NewPassword *string `json:"new_password" binding:"required,min=6"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Token and new_password (min 6 chars) are required"})
			return
		}

		var foundUser models.User
		err := userCollection.FindOne(ctx, bson.M{"reset_token": body.Token}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset link"})
			return
		}
		if foundUser.Reset_expires == nil || foundUser.Reset_expires.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Reset link has expired"})
			return
		}

		hashed := helpers.HashPassword(body.NewPassword)
		_, err = userCollection.UpdateOne(ctx, bson.M{"user_id": foundUser.User_id}, bson.D{
			{"$set", bson.D{
				{"password", *hashed},
				{"reset_token", nil},
				{"reset_expires", nil},
				{"updated_at", time.Now()},
			}},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
	}
}
