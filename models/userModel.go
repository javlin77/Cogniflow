package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	First_name    *string            `json:"first_name" validate:"required,min=2,max=100"`
	Last_name     *string            `json:"last_name" validate:"required,min=2,max=100"`
	Password      *string            `json:"password" validate:"required,min=6"`
	Email         *string            `json:"email" validate:"required,email"`
	Phone         *string            `json:"phone"`
	Token         *string            `json:"token,omitempty"`
	Role          *string            `json:"role"`
	Refresh_token *string            `json:"refresh_token,omitempty"`
	Reset_token   *string            `json:"-" bson:"reset_token,omitempty"`
	Reset_expires *time.Time         `json:"-" bson:"reset_expires,omitempty"`
	Created_at    time.Time          `json:"created_at"`
	Updated_at    time.Time          `json:"updated_at"`
	User_id       string             `json:"user_id"`
}
