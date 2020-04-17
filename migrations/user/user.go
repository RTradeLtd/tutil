package user

import (
	"log"

	"github.com/RTradeLtd/database/v2/models"
	"github.com/jinzhu/gorm"
)

// User provides user migration utlities
type User struct {
	um *models.UserManager
	us *models.UsageManager
}

// NewUserMigration instantiates the user migration tool
func NewUserMigration(db *gorm.DB) *User {
	return &User{
		um: models.NewUserManager(db),
		us: models.NewUsageManager(db),
	}
}

// VerifyUnverifiedUsers is used to verify and upgrade all unverified users
// it returns the number of verified users
func (u *User) VerifyUnverifiedUsers() (int, error) {
	users := []models.User{}
	if err := u.um.DB.Model(&models.User{}).Find(&users).Error; err != nil {
		return 0, err
	}
	var count int
	for _, user := range users {
		if !user.EmailEnabled {
			if _, err := u.um.ValidateEmailVerificationToken(
				user.UserName,
				user.EmailVerificationToken,
			); err != nil {
				log.Println("failed to validate email token for user: ", user.UserName, err.Error())
				continue
			}
		}
		usg, err := u.us.FindByUserName(user.UserName)
		if err != nil {
			log.Println("failed to find usage entry for user: ", user.UserName, err.Error())
			continue
		}
		if usg.Tier == models.Unverified {
			if err := u.us.UpdateTier(user.UserName, models.Free); err != nil {
				log.Println("failed to update tier for user: ", user.UserName, err.Error())
				continue
			}
		}
		log.Println("successfully validate email and upgraded tier for: ", user.UserName)
		count++
	}
	return count, nil
}
