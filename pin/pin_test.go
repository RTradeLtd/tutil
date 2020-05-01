package pin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RTradeLtd/config/v2"
	"github.com/RTradeLtd/database/v2/models"
	"github.com/jinzhu/gorm"
)

const (
	testCID = "QmS4ustL54uo8FzR9455qaxZwuMiUhyvMcX9Ba8nUH4uVv"
)

func TestMigration(t *testing.T) {
	cfg, err := config.LoadConfig("../testenv/config.json")
	if err != nil {
		t.Fatal(err)
	}
	db, err := openDatabaseConnection(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Upload{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Usage{}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestPinExpirationService(t *testing.T) {
	// load configuration
	cfg, err := config.LoadConfig("../testenv/config.json")
	if err != nil {
		t.Fatal(err)
	}
	// open db
	db, err := openDatabaseConnection(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// initialize our pin utility client
	util, err := NewPinUtil(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ue, err := util.US.NewUsageEntry("testuser", models.Paid); err != nil {
		t.Fatal(err)
	} else {
		defer util.US.DB.Unscoped().Delete(ue)
	}
	stats, err := util.ipfs.Stat(testCID)
	if err != nil {
		t.Fatal(err)
	}
	if err := util.US.UpdateDataUsage("testuser", uint64(stats.CumulativeSize)); err != nil {
		t.Fatal(err)
	}
	upload, err := util.UP.NewUpload(testCID, "pin", models.UploadOptions{HoldTimeInMonths: 1, Username: "testuser"})
	if err != nil {
		t.Fatal(err)
	}
	defer util.UP.DB.Unscoped().Delete(upload)
	upload.GarbageCollectDate = time.Now()
	if err := util.UP.DB.Save(upload).Error; err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 2)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if count, err := util.PinExpirationService(ctx, time.Second); err != nil {
		t.Fatal(err)
	} else if count == 0 {
		t.Fatal("no pins removed")
	}
}
func Test_GetExpiredPins(t *testing.T) {
	// load configuration
	cfg, err := config.LoadConfig("../testenv/config.json")
	if err != nil {
		t.Fatal(err)
	}
	// open db
	db, err := openDatabaseConnection(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// initialize our pin utility client
	util, err := NewPinUtil(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ue, err := util.US.NewUsageEntry("testuser", models.Paid); err != nil {
		t.Fatal(err)
	} else {
		defer util.US.DB.Unscoped().Delete(ue)
	}
	stats, err := util.ipfs.Stat(testCID)
	if err != nil {
		t.Fatal(err)
	}
	if err := util.US.UpdateDataUsage("testuser", uint64(stats.CumulativeSize)); err != nil {
		t.Fatal(err)
	}
	upload, err := util.UP.NewUpload(testCID, "pin", models.UploadOptions{HoldTimeInMonths: 1, Username: "testuser"})
	if err != nil {
		t.Fatal(err)
	}
	defer util.UP.DB.Unscoped().Delete(upload)
	upload.GarbageCollectDate = time.Now()
	if err := util.UP.DB.Save(upload).Error; err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 1)
	uploads, err := util.GetExpiredPins()
	if err != nil {
		t.Fatal(err)
	}
	if err := util.ExpirePins(uploads); err != nil {
		t.Fatal(err)
	}
}

func TestPinRemoval(t *testing.T) {
	// load configuration
	cfg, err := config.LoadConfig("../testenv/config.json")
	if err != nil {
		t.Fatal(err)
	}
	// open db
	db, err := openDatabaseConnection(cfg)
	if err != nil {
		t.Fatal(err)
	}
	usageManager := models.NewUsageManager(db)
	// initialize our pin utility client
	util, err := NewPinUtil(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// setup first test user
	user1, err := util.UM.NewUserAccount(
		"testuser1",
		"password123",
		"testuser1@example.org",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer util.UM.DB.Unscoped().Delete(user1)
	usage1, err := usageManager.FindByUserName("testuser1")
	if err != nil {
		t.Fatal(err)
	}
	defer usageManager.DB.Unscoped().Delete(usage1)
	// setup test upload 1
	upload1, err := util.UP.NewUpload(testCID, "file", models.UploadOptions{
		NetworkName:      "public",
		Username:         "testuser1",
		HoldTimeInMonths: 1,
		Encrypted:        false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer util.UP.DB.Unscoped().Delete(upload1)
	if err := util.RemoveAndRefund("testuser", testCID); err != nil {
		t.Fatal(err)
	}
}

func TestPin(t *testing.T) {
	// load configuration
	cfg, err := config.LoadConfig("../testenv/config.json")
	if err != nil {
		t.Fatal(err)
	}
	// open db
	db, err := openDatabaseConnection(cfg)
	if err != nil {
		t.Fatal(err)
	}
	usageManager := models.NewUsageManager(db)
	// initialize our pin utility client
	util, err := NewPinUtil(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// setup first test user
	user1, err := util.UM.NewUserAccount(
		"testuser1",
		"password123",
		"testuser1@example.org",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer util.UM.DB.Unscoped().Delete(user1)
	usage1, err := usageManager.FindByUserName("testuser1")
	if err != nil {
		t.Fatal(err)
	}
	defer usageManager.DB.Unscoped().Delete(usage1)

	// setup second test user
	user2, err := util.UM.NewUserAccount(
		"testuser2",
		"password123",
		"testuser2@example.org",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer util.UM.DB.Unscoped().Delete(user2)
	usage2, err := usageManager.FindByUserName("testuser2")
	if err != nil {
		t.Fatal(err)
	}
	defer usageManager.DB.Unscoped().Delete(usage2)

	// setup test upload 1
	upload1, err := util.UP.NewUpload(testCID, "file", models.UploadOptions{
		NetworkName:      "public",
		Username:         "testuser1",
		HoldTimeInMonths: 1,
		Encrypted:        false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer util.UP.DB.Unscoped().Delete(upload1)

	// setup test upload 2
	upload2, err := util.UP.NewUpload(testCID, "file", models.UploadOptions{
		NetworkName:      "public",
		Username:         "testuser2",
		HoldTimeInMonths: 1,
		Encrypted:        false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer util.UP.DB.Unscoped().Delete(upload2)
	msgs, err := util.GetPinsToRemind(60)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(msgs)
}

func openDatabaseConnection(cfg *config.TemporalConfig) (*gorm.DB, error) {
	dbConnURL := fmt.Sprintf("host=127.0.0.1 port=%s user=postgres dbname=temporal password=%s sslmode=disable",
		cfg.Database.Port, cfg.Database.Password)

	return gorm.Open("postgres", dbConnURL)
}
