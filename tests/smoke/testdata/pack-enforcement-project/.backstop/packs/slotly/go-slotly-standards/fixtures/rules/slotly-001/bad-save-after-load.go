package database

import (
	"gorm.io/gorm"
)

// updateUserBad demonstrates the dangerous pattern: Save after load.
func updateUserBad(db *gorm.DB, userID uint) error {
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		return err
	}
	// user now has decrypted tokens in memory
	user.Timezone = "America/New_York"
	// ruleid: slotly-001
	return db.Save(&user).Error
}

type User struct {
	ID       uint
	Timezone string
}
