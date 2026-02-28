package controllers

import (
	"authentication/helpers"
	"authentication/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateStudySession creates a study session for the current user.
func CreateStudySession() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := getUserID(c)
		if userID == "" {
			return
		}
		var body struct {
			Mode       string `json:"mode"`        // timer | stopwatch
			Goal       string `json:"goal"`        // coding, studying, etc.
			PlannedMin int    `json:"planned_min"` // only for timer
			ActualMin  int    `json:"actual_min"`
			PauseCount int    `json:"pause_count"`
			SelfRating int    `json:"self_rating"`  // 1-5
			SelfOnTask string `json:"self_on_task"` // yes / somewhat / no
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session payload"})
			return
		}
		body.Mode = strings.ToLower(strings.TrimSpace(body.Mode))
		if body.Mode != "timer" && body.Mode != "stopwatch" {
			body.Mode = "stopwatch"
		}
		if strings.TrimSpace(body.Goal) == "" {
			body.Goal = "unspecified"
		}
		if body.ActualMin <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "actual_min must be greater than 0"})
			return
		}
		if body.SelfRating <= 0 {
			body.SelfRating = 3
		}
		if body.PauseCount < 0 {
			body.PauseCount = 0
		}

		session, err := services.CreateStudySession(
			userID,
			body.Mode,
			body.Goal,
			body.PlannedMin,
			body.ActualMin,
			body.PauseCount,
			body.SelfRating,
			body.SelfOnTask,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, session)
	}
}

func getUserID(c *gin.Context) string {
	claimsVal, ok := c.Get("claims")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return ""
	}
	claims, ok := claimsVal.(*helpers.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid claims"})
		return ""
	}
	return claims.UserID
}

// GetMySessions returns study sessions for the current user.
func GetMySessions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := getUserID(c)
		if userID == "" {
			return
		}
		limit := int64(30)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.ParseInt(l, 10, 64); err == nil && n > 0 {
				limit = n
			}
		}
		sessions, err := services.GetSessionsByUser(userID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, sessions)
	}
}

// GetMyFatigueScores returns fatigue scores for the current user.
func GetMyFatigueScores() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := getUserID(c)
		if userID == "" {
			return
		}
		limit := int64(14)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.ParseInt(l, 10, 64); err == nil && n > 0 {
				limit = n
			}
		}
		scores, err := services.GetFatigueScoresByUser(userID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, scores)
	}
}

// GetHighRiskUsers returns top high-risk users (admin only).
func GetHighRiskUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := int64(10)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.ParseInt(l, 10, 64); err == nil && n > 0 {
				limit = n
			}
		}
		scores, err := services.GetHighRiskUsers(limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, scores)
	}
}