package pin

import (
	"fmt"
	"testing"

	"github.com/RTradeLtd/config"
	"github.com/RTradeLtd/database/models"
	"github.com/RTradeLtd/gorm"
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
