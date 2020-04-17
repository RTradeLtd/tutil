package user

import (
	"fmt"
	"testing"

	"github.com/RTradeLtd/config/v2"
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

func TestUserMigration(t *testing.T) {
	cfg, err := config.LoadConfig("../testenv/config.json")
	if err != nil {
		t.Fatal(err)
	}
	db, err := openDatabaseConnection(cfg)
	if err != nil {
		t.Fatal(err)
	}
	testDBMigration(t, db)
	userm := NewUserMigration(db)
	if _, err := userm.um.NewUserAccount(
		"testuser1forusermigration",
		"password123",
		"user1migration@example.org",
	); err != nil {
		t.Fatal(err)
	}
	count, err := userm.VerifyUnverifiedUsers()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("bad number of verified users")
	}
}

func openDatabaseConnection(cfg *config.TemporalConfig) (*gorm.DB, error) {
	dbConnURL := fmt.Sprintf("host=127.0.0.1 port=%s user=postgres dbname=temporal password=%s sslmode=disable",
		cfg.Database.Port, cfg.Database.Password)

	return gorm.Open("postgres", dbConnURL)
}
