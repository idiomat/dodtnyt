package integration

import (
	"gorm.io/gorm"
)

type Author struct {
	gorm.Model
	Name string
}

type Book struct {
	gorm.Model
	Title   string
	Authors []Author `gorm:"many2many:book_authors;"`
}

type LibraryService struct {
	db *gorm.DB
}

func NewLibraryService(db *gorm.DB) *LibraryService {
	return &LibraryService{db: db}
}

func (s *LibraryService) CreateBook(book *Book) error {
	return s.db.Create(book).Error
}

func (s *LibraryService) GetBook(id uint) (*Book, error) {
	var book Book
	err := s.db.Preload("Authors").First(&book, id).Error
	return &book, err
}
