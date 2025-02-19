package models

import (
	"time"
	"fmt"
	"encoding/json"
	"github.com/google/uuid"
)

// CustomDate struct for handling JSON and SQL interactions
type CustomDate struct {
	time.Time
}

// Implement `sql.Scanner` for database retrieval
func (cd *CustomDate) Scan(value interface{}) error {
	if value == nil {
		*cd = CustomDate{Time: time.Time{}} // Set zero time for NULL values
		return nil
	}
	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("cannot scan type %T into CustomDate", value)
	}
	*cd = CustomDate{Time: t}
	return nil
}

// JSON Unmarshaling - Handle Empty and Null Values
func (cd *CustomDate) UnmarshalJSON(data []byte) error {
	var dateStr string
	if err := json.Unmarshal(data, &dateStr); err != nil {
		return err // Return error if invalid format
	}

	if dateStr == "" || dateStr == "null" { // Handle null/empty values
		*cd = CustomDate{}
		return nil
	}

	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return err
	}
	cd.Time = parsedDate
	return nil
}

// JSON Marshaling - Format output as "YYYY-MM-DD"
func (cd CustomDate) MarshalJSON() ([]byte, error) {
	if cd.Time.IsZero() { // Return `null` for zero values
		return []byte("null"), nil
	}
	formatted := fmt.Sprintf("\"%s\"", cd.Time.Format("2006-01-02"))
	return []byte(formatted), nil
}

// Todo struct
type Todo struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	DueDate     CustomDate `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	IsDeleted   bool       `json:"is_deleted"`
}
