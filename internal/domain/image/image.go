package image

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Tags is a string slice stored as JSON in postgres
type Tags []string

func (t Tags) Value() (driver.Value, error) {
	if t == nil {
		return "[]", nil
	}
	b, err := json.Marshal(t)
	return string(b), err
}

func (t *Tags) Scan(value interface{}) error {
	if value == nil {
		*t = Tags{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, t)
}

// Image represents a media library asset stored in Cloudflare R2
type Image struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title      string    `gorm:"size:255;not null" json:"title"`
	AltText    string    `gorm:"size:500" json:"alt_text"`
	URL        string    `gorm:"size:1024;not null" json:"url"`
	ObjectKey  string    `gorm:"size:512;not null" json:"object_key"`
	Size       int64     `gorm:"default:0" json:"size"`
	MimeType   string    `gorm:"size:100" json:"mime_type"`
	Tags       Tags      `gorm:"type:jsonb;default:'[]'" json:"tags"`
	UploadedBy *uint     `gorm:"index" json:"uploaded_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Image) TableName() string {
	return "images"
}
