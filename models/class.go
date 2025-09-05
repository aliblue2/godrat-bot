package models

import "github.com/google/uuid"

type Class struct {
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Master    string    `json:"master"`
	Link      string    `json:"link"`
	Semester  string    `json:"semester"`
	IsPrimary bool      `json:"is_primary"`
}
