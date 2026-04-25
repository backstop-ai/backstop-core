package database

import (
	"gorm.io/gorm"
)

// updateViaPointer demonstrates the dangerous pattern with a pointer variable.
func updateViaPointer(db *gorm.DB, user *User) error {
	user.Timezone = "UTC"
	// ruleid: slotly-001
	return db.Save(user).Error
}

type User struct {
	ID       uint
	Timezone string
}
