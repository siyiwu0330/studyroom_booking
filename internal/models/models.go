package models

type User struct {
	ID      string `json:"id"`       // hex ObjectID
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}
