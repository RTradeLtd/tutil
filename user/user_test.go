package user

import (
	"fmt"
	"testing"

	"github.com/RTradeLtd/config"
	"github.com/RTradeLtd/database/v2/models"
	"github.com/jinzhu/gorm"
)

func testDBMigration(t *testing.T, db *gorm.DB) {
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

func TestUserDelete(t *testing.T) {
	var (
		username1 = "testuserdelete"
		username2 = "testusernodelete"
	)
	cfg, err := config.LoadConfig("../testenv/config.json")
	if err != nil {
		t.Fatal(err)
	}
	db, err := openDatabaseConnection(cfg)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewUserManager(db)
	// START TEST DATA GENERATION
	if usr, err := manager.um.NewUserAccount(username1, "password123", username1+"@example.org"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(usr)
	}
	if usr, err := manager.um.NewUserAccount(username2, "password123", username2+"@example.org"); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(usr)
	}
	if usg, err := manager.us.FindByUserName(username1); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(usg)
	}
	if usg, err := manager.us.FindByUserName(username2); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(usg)
	}
	if up, err := manager.up.NewUpload("testhash1", "file", models.UploadOptions{
		Username:         username1,
		NetworkName:      "public",
		HoldTimeInMonths: 10,
	}); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(up)
	}
	if up, err := manager.up.NewUpload("testhash1", "file", models.UploadOptions{
		Username:         username2,
		NetworkName:      "public",
		HoldTimeInMonths: 10,
	}); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(up)
	}
	if up, err := manager.up.NewUpload("testhash2", "file", models.UploadOptions{
		Username:         username1,
		NetworkName:      "public",
		HoldTimeInMonths: 10,
	}); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(up)
	}
	if up, err := manager.up.NewUpload("testhash2", "file", models.UploadOptions{
		Username:         username2,
		NetworkName:      "public",
		HoldTimeInMonths: 10,
	}); err != nil {
		t.Fatal(err)
	} else {
		defer db.Unscoped().Delete(up)
	}
	// END TEST DATA GENERATION
	if err := manager.Delete(username1); err != nil {
		t.Fatal(err)
	}
	uploads, _ := manager.GetUploads(username1)
	if len(uploads) != 0 {
		t.Fatal("failed to delete uplaods")
	}
	uploads, _ = manager.GetUploads(username2)
	if len(uploads) != 2 {
		t.Fatal("username2 has invalid number of uploads")
	}
}

func openDatabaseConnection(cfg *config.TemporalConfig) (*gorm.DB, error) {
	dbConnURL := fmt.Sprintf("host=127.0.0.1 port=%s user=postgres dbname=temporal password=%s sslmode=disable",
		cfg.Database.Port, cfg.Database.Password)

	return gorm.Open("postgres", dbConnURL)
}
