package user

import (
	"github.com/RTradeLtd/database/v2/models"
	"github.com/RTradeLtd/tutil/utils"
	"github.com/jinzhu/gorm"
)

// User provides user management utlities
type User struct {
	um *models.UserManager
	us *models.UsageManager
	up *models.UploadManager
}

// NewUserManager instantiates the user manager tool tool
func NewUserManager(db *gorm.DB) *User {
	return &User{
		um: models.NewUserManager(db),
		us: models.NewUsageManager(db),
		up: models.NewUploadManager(db),
	}
}

// Deletes a use and cleans out all their data
// replacing with a generic account. helps maintain
// compliance with GDPR
func (u *User) Delete(username string) error {
	usr, err := u.um.FindByUserName(username)
	if err != nil {
		return err
	}
	// get a randomly generated username to indicate user is disabled
	// we set a random username, overwrite the email, and disable ability to login
	newUsername := utils.NewRandomString(200)
	usr.UserName = newUsername
	usr.EmailAddress = newUsername + "@deleteduser.org"
	usr.AccountEnabled = false
	usr.EmailEnabled = false
	if err := u.um.DB.Save(usr).Error; err != nil {
		return err
	}
	usage, err := u.us.FindByUserName(username)
	if err != nil {
		return err
	}
	usage.UserName = newUsername
	if err := u.us.DB.Save(usage).Error; err != nil {
		return err
	}
	uploads, err := u.GetUploads(username)
	if err != nil {
		return err
	}
	for _, upload := range uploads {
		upload.UserName = newUsername
		if err := u.up.DB.Save(upload).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetUploads returns uploads for au ser
func (u *User) GetUploads(username string) ([]models.Upload, error) {
	var uploads []models.Upload
	if err := u.um.DB.Where("user_name = ?", username).Find(&uploads).Error; err != nil {
		return nil, err
	}
	return uploads, nil
}
