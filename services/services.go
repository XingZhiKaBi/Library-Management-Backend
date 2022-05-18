package services

import (
	"errors"
	"github.com/smartwalle/alipay/v3"
	. "gorm.io/gorm"
)

type StatusCode int

type DBAgent struct {
	DB *DB
}

type PayAgent struct {
	PayClient *alipay.Client
}

type Agent struct {
	*DBAgent
	*PayAgent
}

type Book struct {
	Id         int    `column:"id" gorm:"primaryKey"`
	Name       string `column:"name"`
	Author     string `column:"author"`
	Isbn       string `column:"isbn"`
	Language   string `column:"language"`
	LocationId int    `column:"location_id"`
	CategoryId int    `column:"category_id"`
}

type BookMetaData struct {
	Id       int
	Name     string
	Author   string
	Isbn     string
	Language string
	Location string
	Category string
	Status   BookStatus
}

type ReserveBook struct {
	Id        int    `gorm:"column:id;primaryKey"`
	BookId    int    `gorm:"column:book_id"`
	UserId    int    `gorm:"column:user_id"`
	StartTime string `gorm:"column:start_time"`
	EndTime   string `gorm:"column:end_time"`
}

type BorrowBook struct {
	Id        int    `gorm:"column:id;primaryKey"`
	BookId    int    `gorm:"column:book_id"`
	UserId    int    `gorm:"column:user_id"`
	StartTime string `gorm:"column:start_time"`
	EndTime   string `gorm:"column:end_time"`
}

type Location struct {
	Id       int    `gorm:"column:id;primaryKey"`
	Name     string `gorm:"column:name"`
}

type Category struct {
	Id       int    `gorm:"column:id;primaryKey"`
	Name     string `gorm:"column:name"`
}

type BookStatus int

const (
	Idle BookStatus = iota
	Reserved
	Borrowed
)

type ReserveBookStatus struct {
	BookMetaData
	StartTime    string
	EndTime      string
	CanceledTime string
}

type BorrowBookStatus struct {
	BookMetaData
	StartTime string
	EndTime   string
	Deadline  string
	Fine      int
}

type Pay struct {
	Id     int `gorm:"column:id;primaryKey"`
	UserId int `gorm:"column:user_id"`
	Amount int `gorm:"column:amount"`
	Done   int `gorm:"column:done"`
}

type StatusResult struct {
	Code   int
	Msg    string
	Status StatusCode
}

const itemsPerPage = 10

const (
	OK StatusCode = iota
	LoginIdNotExist
	LoginIdOrPasswordError
	LoginOK
	RegisterFailed
	RegisterOK
	ReserveFailed
	ReserveOK
	CancelReserveFailed
	CancelReserveOK
	BorrowFailed
	BorrowOK
	ReturnFailed
	ReturnOK
	AddFailed
	AddOK
	UpdateFailed
	UpdateOK
	DeleteFailed
	DeleteOK
	UpdatePasswordFailed
	UpdatePasswordOK
	DeleteUserFailed
	DeleteUserOK
	AddCategoryFailed
	AddCategoryOK
	AddLocationFailed
	AddLocationOK
)

var (
	MediaPath string
)

func (book *Book) TableName() string {
	return "book"
}

func (reserveBook *ReserveBook) TableName() string {
	return "reserve"
}

func (borrowBook *BorrowBook) TableName() string {
	return "borrow"
}

func (location *Location) TableName() string {
	return "location"
}

func (category *Category) TableName() string {
	return "category"
}

func (pay *Pay) TableName() string {
	return "pay"
}

func (agent *DBAgent) getBookData(book *Book) BookMetaData {
	bookData := BookMetaData{
		Id:       book.Id,
		Name:     book.Name,
		Author:   book.Author,
		Isbn:     book.Isbn,
		Language: book.Language,
		Location: "",
		Category: "",
		Status:   Idle,
	}

	location := Location{}
	agent.DB.First(&location, book.LocationId)
	bookData.Location = location.Name

	category := Category{}
	agent.DB.First(&category, book.CategoryId)
	bookData.Category = category.Name

	if err := agent.DB.Where("book_id = ? and end_time is null", book.Id).Last(&ReserveBook{}).Error; err == nil {
		bookData.Status = Reserved
	}
	if err := agent.DB.Where("book_id = ? and end_time is null", book.Id).Last(&BorrowBook{}).Error; err == nil {
		bookData.Status = Borrowed
	}
	return bookData
}

func (agent *DBAgent) getBooksData(books []Book) []BookMetaData {
	bookDataArr := make([]BookMetaData, 0)
	for _, book := range books {
		bookData := agent.getBookData(&book)
		bookDataArr = append(bookDataArr, bookData)
	}
	return bookDataArr
}

func (agent *DBAgent) GetBooksPages() int64 {
	var count int64
	if err := agent.DB.Table("book").Count(&count).Error; err != nil {
		return 0
	}
	return (count - 1) / itemsPerPage + 1
}

func (agent *DBAgent) GetBooksByPage(page int) []BookMetaData {
	books := make([]Book, 0)
	agent.DB.Offset((page - 1) * 10).Limit(10).Find(&books)
	return agent.getBooksData(books)
}

func (agent *DBAgent) GetBooksPagesByCategory(categoryId int) int64 {
	var count int64
	if err := agent.DB.Table("book").Where("category_id = ?", categoryId).Count(&count).Error; err != nil {
		return 0
	}
	return (count - 1) / itemsPerPage + 1
}

func (agent *DBAgent) GetBooksByCategory(page int, categoryId int) []BookMetaData {
	books := make([]Book, 0)
	agent.DB.Where("category_id = ?", categoryId).Offset((page - 1) * 10).Limit(10).Find(&books)
	return agent.getBooksData(books)
}

func (agent *DBAgent) GetBooksPagesByLocation(locationId int) int64 {
	var count int64
	if err := agent.DB.Table("book").Where("location_id = ?", locationId).Count(&count).Error; err != nil {
		return 0
	}
	return (count - 1) / itemsPerPage + 1
}

func (agent *DBAgent) GetBooksByLocation(page int, locationId int) []BookMetaData {
	books := make([]Book, 0)
	agent.DB.Where("location_id = ?", locationId).Offset((page - 1) * 10).Limit(10).Find(&books)
	return agent.getBooksData(books)
}

func (agent *DBAgent) GetCategories() []Category {
	categories := make([]Category, 0)
	agent.DB.Find(&categories)
	return categories
}

func (agent *DBAgent) GetLocations() []Location {
	locations := make([]Location, 0)
	agent.DB.Find(&locations)
	return locations
}

func (agent *DBAgent) UpdatePassword(userId int, oldPassword string, newPassword string) StatusResult {
	result := StatusResult{}
	err := agent.DB.Transaction(func(tx *DB) error {
		user := User{}
		if err := tx.First(&user, userId).Error; err != nil {
			return err
		}
		if user.Password != oldPassword {
			return errors.New("密码不正确")
		}
		if err := tx.Model(&user).Update("password", newPassword).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		result.Status = UpdatePasswordFailed
		result.Msg = "修改密码失败"
		return result
	}
	result.Status = UpdatePasswordOK
	result.Msg = "修改密码成功"
	return result
}
