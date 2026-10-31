package models

import (
	"time"
)

// KondisiBarang - enum untuk kondisi barang
const (
	KondisiBaik        = "Baik"
	KondisiRusakRingan = "Rusak Ringan"
	KondisiRusakBerat  = "Rusak Berat"
)

// Role - enum untuk role user
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
)

// User - model untuk user authentication
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	Role      string    `gorm:"size:20;not null;default:operator" json:"role"`
	FullName  string    `gorm:"size:100" json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Room - model untuk ruangan
type Room struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:20;not null" json:"code"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Item - model untuk barang inventaris
type Item struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Code          string     `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Name          string     `gorm:"size:200;not null" json:"name"`
	RoomID        uint       `gorm:"not null" json:"room_id"`
	Room          *Room      `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Condition     string     `gorm:"size:30;not null;default:Baik" json:"condition"`
	AcquisitionDate *time.Time `json:"acquisition_date,omitempty"`
	Price         float64    `gorm:"type:decimal(15,2);default:0" json:"price"`
	Description   string     `gorm:"size:500" json:"description"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TransferHistory - model untuk riwayat pemindahan barang
type TransferHistory struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ItemID        uint      `gorm:"not null;index" json:"item_id"`
	Item          *Item     `gorm:"foreignKey:ItemID" json:"item,omitempty"`
	FromRoomID    uint      `json:"from_room_id"`
	FromRoom      *Room     `gorm:"foreignKey:FromRoomID" json:"from_room,omitempty"`
	ToRoomID      uint      `gorm:"not null" json:"to_room_id"`
	ToRoom        *Room     `gorm:"foreignKey:ToRoomID" json:"to_room,omitempty"`
	UserID        uint      `gorm:"not null" json:"user_id"`
	User          *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Reason        string    `gorm:"size:500" json:"reason"`
	TransferDate  time.Time `gorm:"not null" json:"transfer_date"`
	CreatedAt     time.Time `json:"created_at"`
}

// Notification - model untuk notifikasi ke admin
type Notification struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Type      string     `gorm:"size:50;not null;index" json:"type"`
	Title     string     `gorm:"size:200;not null" json:"title"`
	Message   string     `gorm:"size:1000;not null" json:"message"`
	ItemID    *uint      `gorm:"index" json:"item_id,omitempty"`
	Item      *Item      `gorm:"foreignKey:ItemID" json:"item,omitempty"`
	UserID    *uint      `json:"user_id,omitempty"`
	User      *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	IsRead    bool       `gorm:"default:false;index" json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

// ItemConditionHistory - model untuk riwayat perubahan kondisi barang
type ItemConditionHistory struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ItemID       uint      `gorm:"not null;index" json:"item_id"`
	Item         *Item     `gorm:"foreignKey:ItemID" json:"item,omitempty"`
	OldCondition string    `gorm:"size:30" json:"old_condition"`
	NewCondition string    `gorm:"size:30;not null" json:"new_condition"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	User         *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Note         string    `gorm:"size:500" json:"note"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// LoginRequest - request body untuk login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse - response untuk login
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ItemRequest - request body untuk create/update item
type ItemRequest struct {
	Code            string  `json:"code" binding:"required"`
	Name            string  `json:"name" binding:"required"`
	RoomID          uint    `json:"room_id" binding:"required"`
	Condition       string  `json:"condition"`
	AcquisitionDate string  `json:"acquisition_date"`
	Price           float64 `json:"price"`
	Description     string  `json:"description"`
}

// RoomRequest - request body untuk create/update room
type RoomRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// TransferRequest - request body untuk pemindahan barang
type TransferRequest struct {
	ItemID       uint   `json:"item_id" binding:"required"`
	ToRoomID     uint   `json:"to_room_id" binding:"required"`
	Reason       string `json:"reason"`
	TransferDate string `json:"transfer_date"`
}

// ScanRequest - request body untuk scan QR dari mobile
type ScanRequest struct {
	Code      string   `json:"code" binding:"required"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// UpdateConditionRequest - request body untuk update kondisi dari scan
type UpdateConditionRequest struct {
	Code        string  `json:"code" binding:"required"`
	Condition   string  `json:"condition" binding:"required"`
	Note        string  `json:"note"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
}

// UserRequest - request body untuk create user
type UserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
	FullName string `json:"full_name"`
}

// APIResponse - standard response format
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
