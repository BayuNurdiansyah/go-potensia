package config

import (
	"log"
	"os"

	"go-potensia/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.MentorProfile{},
		&models.MentorCertificate{},
		&models.MentorAchievement{},
		&models.MentorGallery{},
		&models.MentorEducation{},
		&models.ParentProfile{},
		&models.Child{},
		&models.Course{},
		&models.CourseCompetency{},
		&models.CoursePackage{},
		&models.Order{},
		&models.Session{},
		&models.Invoice{},
		&models.Review{},
		&models.Notification{},
		&models.SkillProgress{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	DB = db
	log.Println("✅ Database connected and migrated")
}