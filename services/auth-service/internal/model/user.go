package model

import "time"

type User struct {
	ID            string    `json:"id"`
	PhoneNumber   string    `json:"phone_number"`
	Status        string    `json:"status"`
	PhoneVerified bool      `json:"phone_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
