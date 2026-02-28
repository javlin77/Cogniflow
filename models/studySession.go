package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StudySession struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	Mode        string             `bson:"mode" json:"mode"`                         // "timer" or "stopwatch"
	Goal        string             `bson:"goal" json:"goal"`                         // e.g. coding, studying
	PlannedMin  int                `bson:"planned_min,omitempty" json:"planned_min,omitempty"`
	StartedAt   time.Time          `bson:"started_at" json:"started_at"`
	EndedAt     *time.Time         `bson:"ended_at,omitempty" json:"ended_at,omitempty"`
	DurationMin int                `bson:"duration_min" json:"duration_min"` // actual in minutes
	FocusScore  int                `bson:"focus_score" json:"focus_score"`   // 0-100 AI-computed
	PauseCount  int                `bson:"pause_count" json:"pause_count"`
	SelfRating  int                `bson:"self_rating" json:"self_rating"`       // 1-5 self-assessed
	SelfOnTask  string             `bson:"self_on_task" json:"self_on_task"`     // yes / somewhat / no
	Breaks      []BreakInterval    `bson:"breaks,omitempty" json:"breaks,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

type BreakInterval struct {
	StartedAt time.Time `bson:"started_at" json:"started_at"`
	EndedAt   time.Time `bson:"ended_at" json:"ended_at"`
	Minutes   int      `bson:"minutes" json:"minutes"`
}
