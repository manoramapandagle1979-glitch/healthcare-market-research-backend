package author

import (
	"time"
)

// Author represents an author/analyst who writes reports
type Author struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	Role      string    `json:"role,omitempty" gorm:"type:varchar(100)"`
	Bio       string    `json:"bio,omitempty" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName specifies the table name for GORM
func (Author) TableName() string {
	return "authors"
}
