package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FatigueScore stores computed fatigue metrics per user (e.g. daily rollup).
type FatigueScore struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	UserID            string             `bson:"user_id" json:"user_id"`
	Date              time.Time          `bson:"date" json:"date"` // day (UTC midnight)
	TotalStudyHours   float64            `bson:"total_study_hours" json:"total_study_hours"`
	BreakFrequency    float64            `bson:"break_frequency" json:"break_frequency"`       // breaks per hour
	FocusStability    float64            `bson:"focus_stability" json:"focus_stability"`       // 0-100, lower = more volatile
	FatigueIndex      float64            `bson:"fatigue_index" json:"fatigue_index"`          // weighted composite
	BurnoutProbability float64           `bson:"burnout_probability" json:"burnout_probability"` // 0-100
	CreatedAt         time.Time         `bson:"created_at" json:"created_at"`
}
