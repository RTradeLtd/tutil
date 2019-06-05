package pin

import (
	"errors"
	"fmt"
	"time"

	"github.com/RTradeLtd/config/v2"
	"github.com/RTradeLtd/database/v2/models"
	"github.com/RTradeLtd/gorm"
	"github.com/RTradeLtd/rtfs/v2"
	"github.com/RTradeLtd/tutil/mail"
)

// Util is our pin related utility class
type Util struct {
	UM   *models.UserManager
	UP   *models.UploadManager
	US   *models.UsageManager
	ipfs rtfs.Manager
	Mail *mail.Manager
}

// NewPinUtil is used to generate our pin related utilities
func NewPinUtil(db *gorm.DB, cfg *config.TemporalConfig) (*Util, error) {
	manager, err := mail.NewManager(cfg, db)
	if err != nil {
		return nil, err
	}
	ipfs, err := rtfs.NewManager(
		cfg.IPFS.APIConnection.Host+":"+cfg.IPFS.APIConnection.Port,
		"", time.Hour,
	)
	if err != nil {
		return nil, err
	}
	return &Util{
		UM:   models.NewUserManager(db),
		UP:   models.NewUploadManager(db),
		US:   models.NewUsageManager(db),
		ipfs: ipfs,
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

// GetExpiredPins is used to retrieve all uploads/pins
// that are currently expired and need to be removed
func (u *Util) GetExpiredPins() ([]models.Upload, error) {
	uploads := []models.Upload{}
	currentDate := time.Now()
	if err := u.UP.DB.Model(&models.Upload{}).Where(
		"garbage_collect_date < ?", currentDate,
	).Find(&uploads).Error; err != nil {
		return nil, err
	}
	if len(uploads) == 0 {
		return nil, errors.New("no expired pins")
	}
	return uploads, nil
}

// ExpirePins is used to remove all expired pins from
// a users given uploads, as well as reducing their data usage
func (u *Util) ExpirePins(uploads []models.Upload) error {
	for _, upload := range uploads {
		// get variables needed for filtration
		hash := upload.Hash
		user := upload.UserName
		stats, err := u.ipfs.Stat(hash)
		if err != nil {
			fmt.Printf(
				"failed to get object stats for hash %s, user %s. error: %s",
				hash, user, err.Error(),
			)
			continue
		}
		sizeToReduce := uint64(stats.CumulativeSize)
		if err := u.US.ReduceDataUsage(user, sizeToReduce); err != nil {
			fmt.Printf(
				"failed to reduce data usage for hash %s, user %s. error: %s",
				hash, user, err.Error(),
			)
			continue
		}
	}
	return nil
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
	// calculate the time window
	maxGCDate := time.Now().AddDate(0, 0, days)
	// find all uploads within the garbage collect period
	if err := u.UP.DB.Model(&models.Upload{}).Where(
		"garbage_collect_date BETWEEN ? AND ?",
		time.Now(), maxGCDate,
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
		var hashFormatted string
		for _, h := range v {
			hashFormatted = hashFormatted + "," + h
		}
		message := fmt.Sprintf(
			"The following hashes you have uploaded will be removed from the system within the next %v days, please extend your pin soon or they will be removed <br>%s",
			days, hashFormatted,
		)
		reminders = append(reminders, ReminderMessage{
			EmailAddress: user.EmailAddress,
			UserName:     user.UserName,
			Message:      message,
		})
	}
	return reminders, nil
}
