package pin

import (
	"fmt"
	"time"

	"github.com/RTradeLtd/config"
	"github.com/RTradeLtd/database/models"
	"github.com/RTradeLtd/gorm"
	"github.com/RTradeLtd/tutil/mail"
)

// Util is our pin related utility class
type Util struct {
	UM   *models.UserManager
	UP   *models.UploadManager
	Mail *mail.Manager
}

// NewPinUtil is used to generate our pin related utilities
func NewPinUtil(db *gorm.DB, cfg *config.TemporalConfig) (*Util, error) {
	manager, err := mail.NewManager(cfg, db)
	if err != nil {
		return nil, err
	}
	return &Util{
		UM:   models.NewUserManager(db),
		UP:   models.NewUploadManager(db),
		Mail: manager,
	}, nil
}

// ReminderMessage is a bulk aggregation of all hashes that will expire soon.
// This allows us to send a single email, without having to spam the user
type ReminderMessage struct {
	EmailAddress string
	UserName     string
	Message      string
}

// GetPinsToRemind is used to get pins that are close to their gc date
// these pins are then used to send an email reminder to the user to remind them
// that they will need to extend the lifetime, otherwise their data will be removed.
//
// the window is the time window we use to examine uploads for reminder. If you give todays date + 7 days
// then we will search for all uploads that expire between now, and 7 days from now.
//
// Notes about when multiple users have pinned the same file:
// if the same file is pinned by multiple users, we won't actually remove it from our system.
// However in the event that the final user who is pinning the content lets the garbage collection date
// expire, then and only then is the data removed from our system.
func (u *Util) GetPinsToRemind(days int) ([]ReminderMessage, error) {
	uploads := []models.Upload{}

	// get the number of hours multiplied by the number of days
	hours := days * 24
	// turn into a time.Time type
	tt := time.Duration(hours) * time.Hour
	// calculate the time window
	maxGCDate := time.Now().Add(tt)
	// grab current time
	currentTime := time.Now()

	// find all uploads within the garbage collect period
	if err := u.UP.DB.Where(
		"garbage_collect_date < ? AND garbage_collect_date > ?",
		maxGCDate, currentTime,
	).Find(&uploads).Error; err != nil {
		return nil, err
	}
	// hashes will hold all hashes belonging to a give user
	hashes := make(map[string][]string)
	// iterate through all uploads to updated the hashes map
	for _, v := range uploads {
		hashes[v.UserName] = append(hashes[v.UserName], v.Hash)
	}
	// a single ReminderMessage will be used to send a single email
	// while also containing all hashes that are going to expire
	reminders := []ReminderMessage{}
	for k, v := range hashes {
		user, err := u.UM.FindByUserName(k)
		if err != nil {
			return nil, err
		}
		message := fmt.Sprintf(
			"The following hashes you have uploaded will be removed from the system within the next %v days, please extend your pin soon or they will be removed <br>%s",
			days, v,
		)
		reminders = append(reminders, ReminderMessage{
			EmailAddress: user.EmailAddress,
			UserName:     user.UserName,
			Message:      message,
		})
	}
	return reminders, nil
}
