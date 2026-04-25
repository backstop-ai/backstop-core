package database

import (
	"fmt"

	"gorm.io/gorm"
)

// updateUserTimezone demonstrates the correct pattern: targeted field update.
func updateUserTimezone(db *gorm.DB, userID uint, tz string) error {
	result := db.Model(&User{}).Where("id = ?", userID).Update("timezone", tz)
	if result.Error != nil {
		return fmt.Errorf("failed to update timezone: %w", result.Error)
	}
	return nil
}

// updateUserPreferences demonstrates correct multi-field targeted update.
func updateUserPreferences(db *gorm.DB, userID uint, start, end int) error {
	updates := map[string]interface{}{
		"schedulable_hours_start": start,
		"schedulable_hours_end":   end,
	}
	result := db.Model(&User{}).Where("id = ?", userID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update preferences: %w", result.Error)
	}
	return nil
}

type User struct {
	ID uint
}
