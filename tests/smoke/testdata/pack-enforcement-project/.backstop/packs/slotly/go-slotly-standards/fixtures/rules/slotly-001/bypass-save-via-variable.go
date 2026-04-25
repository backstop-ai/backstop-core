package database

import (
	"gorm.io/gorm"
)

// bypassAttempt: reassigns the user to a differently-named variable, then
// calls Save on that variable. The intent is to dodge a naive grep for
// "Save(&user)" but the semantic pattern is identical — the struct still
// contains decrypted tokens from a prior load.
func updateViaAlias(db *gorm.DB, userID uint) error {
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		return err
	}
	target := user
	target.Timezone = "Europe/London"
	// ruleid: slotly-001
	return db.Save(&target).Error
}

type User struct {
	ID       uint
	Timezone string
}
