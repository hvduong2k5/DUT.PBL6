package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
)

type MediaRepository struct {
	pool *pgxpool.Pool
}

func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{pool: pool}
}

func (r *MediaRepository) CreatePending(ctx context.Context, record domain.CreateMediaRecord) (domain.MediaFile, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return domain.MediaFile{}, false, fmt.Errorf("begin create media: %w", err)
	}
	defer tx.Rollback(ctx)

	resourceID, created, err := claimIdempotency(
		ctx, tx, record.File.OwnerID, "media.create-upload-session", record.IdempotencyKey,
		record.RequestHash, record.IdempotencyExpiry,
	)
	if err != nil {
		return domain.MediaFile{}, false, err
	}
	if !created {
		id, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return domain.MediaFile{}, false, fmt.Errorf("parse idempotent media ID: %w", parseErr)
		}
		file, getErr := getMedia(ctx, tx, id, record.File.OwnerID)
		if getErr != nil {
			return domain.MediaFile{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.MediaFile{}, false, err
		}
		return file, false, nil
	}

	file := record.File
	_, err = tx.Exec(ctx, `
		INSERT INTO media_files (
			id, owner_id, purpose, provider, bucket, object_key, original_filename,
			declared_mime_type, declared_size_bytes, checksum_sha256, status,
			scan_status, version, created_at, updated_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, NULLIF($10, ''), $11,
			$12, $13, $14, $14, $15
		)
	`, file.ID, file.OwnerID, file.Purpose, file.Provider, file.Bucket, file.ObjectKey,
		file.OriginalFilename, file.DeclaredMIMEType, file.DeclaredSize, file.ChecksumSHA256,
		file.Status, file.ScanStatus, file.Version, file.CreatedAt, file.ExpiresAt)
	if err != nil {
		return domain.MediaFile{}, false, fmt.Errorf("insert media: %w", err)
	}
	if err := completeIdempotency(ctx, tx, file.OwnerID, "media.create-upload-session", record.IdempotencyKey, file.ID.String(), 201); err != nil {
		return domain.MediaFile{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.MediaFile{}, false, fmt.Errorf("commit create media: %w", err)
	}
	return file, true, nil
}

func (r *MediaRepository) GetOwned(ctx context.Context, id, ownerID uuid.UUID) (domain.MediaFile, error) {
	return getMedia(ctx, r.pool, id, ownerID)
}

func (r *MediaRepository) MarkReady(ctx context.Context, id, ownerID uuid.UUID, object domain.StoredObject) (domain.MediaFile, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return domain.MediaFile{}, err
	}
	defer tx.Rollback(ctx)

	file, err := getMediaForUpdate(ctx, tx, id, ownerID)
	if err != nil {
		return domain.MediaFile{}, err
	}
	if file.Status == domain.MediaStatusReady || file.Status == domain.MediaStatusAttached {
		if err := tx.Commit(ctx); err != nil {
			return domain.MediaFile{}, err
		}
		return file, nil
	}
	if file.Status != domain.MediaStatusPendingUpload && file.Status != domain.MediaStatusUploaded && file.Status != domain.MediaStatusScanning {
		return domain.MediaFile{}, domain.ErrMediaNotReady
	}

	row := tx.QueryRow(ctx, `
		UPDATE media_files
		SET detected_mime_type = $3,
			actual_size_bytes = $4,
			storage_etag = $5,
			status = 'READY',
			scan_status = 'CLEAN',
			uploaded_at = COALESCE(uploaded_at, NOW()),
			updated_at = NOW(),
			version = version + 1
		WHERE id = $1 AND owner_id = $2
		RETURNING `+mediaColumns+`
	`, id, ownerID, object.ContentType, object.Size, object.ETag)
	ready, err := scanMedia(row)
	if err != nil {
		return domain.MediaFile{}, fmt.Errorf("mark media ready: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO media_scan_results (id, media_id, scanner, result)
		VALUES ($1, $2, 'configured-scanner', 'CLEAN')
	`, idgen.New(), id)
	if err != nil {
		return domain.MediaFile{}, fmt.Errorf("record media scan: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.MediaFile{}, err
	}
	return ready, nil
}

func (r *MediaRepository) DeleteUnattached(ctx context.Context, id, ownerID uuid.UUID) (domain.MediaFile, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE media_files
		SET status = 'DELETED', deleted_at = NOW(), updated_at = NOW(), version = version + 1
		WHERE id = $1 AND owner_id = $2
		  AND status IN ('PENDING_UPLOAD', 'UPLOADED', 'SCANNING', 'READY')
		RETURNING `+mediaColumns+`
	`, id, ownerID)
	file, err := scanMedia(row)
	if err == pgx.ErrNoRows {
		return domain.MediaFile{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MediaFile{}, fmt.Errorf("delete media: %w", err)
	}
	return file, nil
}

const mediaColumns = `
	id, owner_id, purpose, provider, bucket, object_key, original_filename,
	declared_mime_type, COALESCE(detected_mime_type, ''), declared_size_bytes,
	COALESCE(actual_size_bytes, 0), COALESCE(checksum_sha256, ''), COALESCE(storage_etag, ''),
	status, scan_status, version, created_at, updated_at, uploaded_at, attached_at, expires_at, deleted_at
`

type rowScanner interface {
	Scan(...any) error
}

func scanMedia(row rowScanner) (domain.MediaFile, error) {
	var file domain.MediaFile
	err := row.Scan(
		&file.ID, &file.OwnerID, &file.Purpose, &file.Provider, &file.Bucket, &file.ObjectKey,
		&file.OriginalFilename, &file.DeclaredMIMEType, &file.DetectedMIMEType, &file.DeclaredSize,
		&file.ActualSize, &file.ChecksumSHA256, &file.StorageETag, &file.Status, &file.ScanStatus,
		&file.Version, &file.CreatedAt, &file.UpdatedAt, &file.UploadedAt, &file.AttachedAt,
		&file.ExpiresAt, &file.DeletedAt,
	)
	return file, err
}

type mediaQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getMedia(ctx context.Context, db mediaQuerier, id, ownerID uuid.UUID) (domain.MediaFile, error) {
	file, err := scanMedia(db.QueryRow(ctx, `
		SELECT `+mediaColumns+` FROM media_files WHERE id = $1 AND owner_id = $2
	`, id, ownerID))
	if err == pgx.ErrNoRows {
		return domain.MediaFile{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MediaFile{}, fmt.Errorf("get media: %w", err)
	}
	return file, nil
}

func getMediaForUpdate(ctx context.Context, tx pgx.Tx, id, ownerID uuid.UUID) (domain.MediaFile, error) {
	file, err := scanMedia(tx.QueryRow(ctx, `
		SELECT `+mediaColumns+` FROM media_files WHERE id = $1 AND owner_id = $2 FOR UPDATE
	`, id, ownerID))
	if err == pgx.ErrNoRows {
		return domain.MediaFile{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MediaFile{}, fmt.Errorf("lock media: %w", err)
	}
	return file, nil
}
