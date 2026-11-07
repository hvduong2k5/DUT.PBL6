package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UploadHeaders map[string]string

type StoredObject struct {
	Size           int64
	ContentType    string
	ETag           string
	ChecksumSHA256 string
}

type ObjectStorage interface {
	EnsureBucket(context.Context) error
	Health(context.Context) error
	PresignUpload(context.Context, string, string, string, time.Duration) (string, UploadHeaders, error)
	Stat(context.Context, string) (StoredObject, error)
	PresignDownload(context.Context, string, time.Duration) (string, error)
	Remove(context.Context, string) error
}

type MalwareScanner interface {
	Scan(context.Context, string) error
}

type CreateMediaRecord struct {
	File              MediaFile
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
}

type MediaRepository interface {
	CreatePending(context.Context, CreateMediaRecord) (MediaFile, bool, error)
	GetOwned(context.Context, uuid.UUID, uuid.UUID) (MediaFile, error)
	MarkReady(context.Context, uuid.UUID, uuid.UUID, StoredObject) (MediaFile, error)
	DeleteUnattached(context.Context, uuid.UUID, uuid.UUID) (MediaFile, error)
}

type CreateTicketRecord struct {
	Ticket            SupportTicket
	FirstMessage      TicketMessage
	FileIDs           []uuid.UUID
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
}

type AddTicketMessageRecord struct {
	Message           TicketMessage
	CustomerID        uuid.UUID
	FileIDs           []uuid.UUID
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
}

type TicketPage struct {
	Items      []SupportTicket
	NextCursor string
}

type TicketRepository interface {
	Create(context.Context, CreateTicketRecord) (TicketDetail, bool, error)
	ListOwned(context.Context, uuid.UUID, int, string) (TicketPage, error)
	GetOwned(context.Context, uuid.UUID, uuid.UUID) (TicketDetail, error)
	AddCustomerMessage(context.Context, AddTicketMessageRecord) (TicketMessage, bool, error)
}
