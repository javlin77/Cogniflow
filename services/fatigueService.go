package services

import (
	"authentication/config"
	"authentication/models"
	"context"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	weightStudyHours      = 0.4
	weightBreakFreq       = -0.3 // more breaks = lower fatigue
	weightFocusVolatility = 0.3
)

// -------- Cognitive fatigue helpers --------

// FatigueIndex = (w1 * totalStudyHours) - (w2 * breakFrequency) + (w3 * focusVolatility)
// focusVolatility = 100 - focusStability (higher volatility = higher fatigue)
func ComputeFatigueIndex(totalStudyHours, breakFreqPerHour, focusStability float64) float64 {
	focusVolatility := 100 - focusStability
	if focusVolatility < 0 {
		focusVolatility = 0
	}
	idx := (weightStudyHours * totalStudyHours) - (weightBreakFreq * breakFreqPerHour) + (weightFocusVolatility * focusVolatility/100)
	return math.Max(0, math.Min(100, idx*25)) // scale to rough 0-100
}

// BurnoutProbability: simple logistic-style from fatigue index
func ComputeBurnoutProbability(fatigueIndex float64) float64 {
	p := 100 / (1 + math.Exp(-0.1*(fatigueIndex-50)))
	return math.Round(p*10) / 10
}

// baselineMinutesForGoal returns a reasonable target duration for a session.
func baselineMinutesForGoal(goal, mode string) int {
	g := strings.ToLower(strings.TrimSpace(goal))
	switch g {
	case "coding", "problem solving", "problem_solving":
		return 45
	case "studying", "reading", "research", "revision":
		return 30
	case "writing", "content creation", "content_creation", "video editing", "video_editing":
		return 40
	case "office work", "office_work", "admin":
		return 20
	default:
		// Generic focus block
		if mode == "stopwatch" {
			return 25
		}
		return 25
	}
}

func completionRatio(mode string, plannedMin, actualMin, baseline int) float64 {
	if actualMin <= 0 {
		return 0
	}
	var target float64
	if mode == "timer" && plannedMin > 0 {
		target = float64(plannedMin)
	} else {
		target = float64(baseline)
	}
	if target <= 0 {
		target = float64(actualMin)
	}
	r := float64(actualMin) / target
	if r < 0 {
		r = 0
	}
	if r > 1.2 {
		r = 1.2
	}
	return r / 1.2
}

func stabilityFromPauses(pauseCount int) float64 {
	switch {
	case pauseCount <= 0:
		return 1
	case pauseCount <= 2:
		return 0.8
	case pauseCount <= 4:
		return 0.5
	default:
		return 0.3
	}
}

func selfComponent(selfRating int, selfOnTask string) float64 {
	if selfRating < 1 {
		selfRating = 1
	}
	if selfRating > 5 {
		selfRating = 5
	}
	base := float64(selfRating) / 5.0
	mult := 0.8
	switch strings.ToLower(strings.TrimSpace(selfOnTask)) {
	case "yes", "y", "on_task":
		mult = 1
	case "somewhat":
		mult = 0.7
	case "no", "n":
		mult = 0.4
	}
	v := base * mult
	if v > 1 {
		v = 1
	}
	if v < 0 {
		v = 0
	}
	return v
}

// historicalConsistency looks at recent sessions to reward regular use.
func historicalConsistency(userID string) float64 {
	sessions, err := GetSessionsByUser(userID, 20)
	if err != nil || len(sessions) == 0 {
		return 0.3
	}
	cutoff := time.Now().AddDate(0, 0, -14)
	days := make(map[string]struct{})
	for _, s := range sessions {
		if s.StartedAt.Before(cutoff) {
			continue
		}
		day := s.StartedAt.Format("2006-01-02")
		days[day] = struct{}{}
	}
	unique := len(days)
	if unique == 0 {
		return 0.4
	}
	if unique >= 10 {
		return 1
	}
	return float64(unique) / 10.0
}

// ComputeSessionFocusScore combines multiple factors into a 0–100 score.
func ComputeSessionFocusScore(mode, goal string, plannedMin, actualMin, pauseCount, selfRating int, selfOnTask string, histConsistency float64) int {
	mode = strings.ToLower(mode)
	baseline := baselineMinutesForGoal(goal, mode)
	comp := completionRatio(mode, plannedMin, actualMin, baseline)
	stab := stabilityFromPauses(pauseCount)
	self := selfComponent(selfRating, selfOnTask)
	hc := histConsistency
	if hc < 0 {
		hc = 0
	}
	if hc > 1 {
		hc = 1
	}
	// FocusScore = 0.4 × CompletionRatio + 0.2 × StabilityScore + 0.2 × SelfRating + 0.2 × HistoricalConsistency
	score01 := 0.4*comp + 0.2*stab + 0.2*self + 0.2*hc
	if score01 < 0 {
		score01 = 0
	}
	if score01 > 1 {
		score01 = 1
	}
	return int(math.Round(score01 * 100))
}

func CreateStudySession(userID, mode, goal string, plannedMin, actualMin, pauseCount, selfRating int, selfOnTask string) (*models.StudySession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coll := config.OpenCollection("study_sessions")
	now := time.Now()

	hist := historicalConsistency(userID)
	focusScore := ComputeSessionFocusScore(mode, goal, plannedMin, actualMin, pauseCount, selfRating, selfOnTask, hist)

	s := &models.StudySession{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Mode:        strings.ToLower(mode),
		Goal:        goal,
		PlannedMin:  plannedMin,
		StartedAt:   now,
		DurationMin: actualMin,
		FocusScore:  focusScore,
		PauseCount:  pauseCount,
		SelfRating:  selfRating,
		SelfOnTask:  selfOnTask,
		CreatedAt:   now,
	}
	_, err := coll.InsertOne(ctx, s)
	return s, err
}

func GetSessionsByUser(userID string, limit int64) ([]models.StudySession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coll := config.OpenCollection("study_sessions")
	opts := options.Find().SetSort(bson.D{{"created_at", -1}}).SetLimit(limit)
	cursor, err := coll.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []models.StudySession
	err = cursor.All(ctx, &out)
	return out, err
}

func GetFatigueScoresByUser(userID string, limit int64) ([]models.FatigueScore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coll := config.OpenCollection("fatigue_scores")
	opts := options.Find().SetSort(bson.D{{"date", -1}}).SetLimit(limit)
	cursor, err := coll.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []models.FatigueScore
	err = cursor.All(ctx, &out)
	return out, err
}

func RecomputeAndUpsertFatigueScore(userID string, date time.Time, totalStudyHours, breakFreq, focusStability float64) (*models.FatigueScore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coll := config.OpenCollection("fatigue_scores")
	fatigueIndex := ComputeFatigueIndex(totalStudyHours, breakFreq, focusStability)
	burnoutProb := ComputeBurnoutProbability(fatigueIndex)
	score := &models.FatigueScore{
		ID:                 primitive.NewObjectID(),
		UserID:             userID,
		Date:               date,
		TotalStudyHours:    totalStudyHours,
		BreakFrequency:     breakFreq,
		FocusStability:     focusStability,
		FatigueIndex:       fatigueIndex,
		BurnoutProbability: burnoutProb,
		CreatedAt:          time.Now(),
	}
	filter := bson.M{"user_id": userID, "date": date}
	update := bson.D{{"$set", bson.D{
		{"total_study_hours", score.TotalStudyHours},
		{"break_frequency", score.BreakFrequency},
		{"focus_stability", score.FocusStability},
		{"fatigue_index", score.FatigueIndex},
		{"burnout_probability", score.BurnoutProbability},
		{"created_at", score.CreatedAt},
	}}}
	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, err
	}
	return score, nil
}

// Admin: high-risk users (by latest burnout probability)
func GetHighRiskUsers(limit int64) ([]models.FatigueScore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coll := config.OpenCollection("fatigue_scores")
	// Get latest score per user, then sort by burnout_probability desc
	pipe := []bson.M{
		{"$sort": bson.M{"date": -1}},
		{"$group": bson.M{
			"_id": "$user_id",
			"doc": bson.M{"$first": "$$ROOT"},
		}},
		{"$replaceRoot": bson.M{"newRoot": "$doc"}},
		{"$sort": bson.M{"burnout_probability": -1}},
		{"$limit": limit},
	}
	cursor, err := coll.Aggregate(ctx, pipe)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []models.FatigueScore
	err = cursor.All(ctx, &out)
	return out, err
}
