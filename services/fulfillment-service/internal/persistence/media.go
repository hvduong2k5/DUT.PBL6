package persistence

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"time"
)

func validateMedia(ctx context.Context, tx pgx.Tx, t d.Task, m d.Mutation, now time.Time) error {
	if m.EvidenceID == nil {
		if t.Evidence == "VIDEO_REQUIRED" || t.Cohort == "INTEGRATED_V2" {
			return d.Fail("MEDIA_REQUIRED", 409)
		}
		return nil
	}
	if !d.UUID(*m.EvidenceID) {
		return d.Fail("INVALID_MEDIA_ID", 400)
	}
	var status, task, pkg, scan string
	var key, hash, version *string
	var verified *time.Time
	e := tx.QueryRow(ctx, `SELECT status,task_id::text,package_id::text,scan_status,final_object_key,verified_sha256,object_version,verified_at FROM fulfillment.media_files WHERE file_id=$1 FOR UPDATE`, *m.EvidenceID).Scan(&status, &task, &pkg, &scan, &key, &hash, &version, &verified)
	if e != nil {
		return mapped(e)
	}
	if task != t.ID || pkg != t.Package.ID || status != "VERIFIED" || scan != "SAFE" || key == nil || hash == nil || version == nil || verified == nil {
		return d.Fail("MEDIA_NOT_VERIFIED", 409)
	}
	return nil
}
func (s *Store) File(ctx context.Context, id string) (f d.File, e error) {
	if !d.UUID(id) {
		return f, d.Fail("INVALID_FILE_ID", 400)
	}
	e = s.DB.QueryRow(ctx, `SELECT file_id::text,task_id::text,package_id::text,status,media_type,declared_size_bytes,verified_sha256,verified_at,uploader_id::text,object_key,COALESCE(final_object_key,''),COALESCE(object_version,''),declared_sha256,scanner_policy_version,session_expires_at,version FROM fulfillment.media_files WHERE file_id=$1`, id).Scan(&f.ID, &f.TaskID, &f.PackageID, &f.Status, &f.Type, &f.Size, &f.Checksum, &f.VerifiedAt, &f.Uploader, &f.StagingKey, &f.FinalKey, &f.ObjectVersion, &f.DeclaredHash, &f.ScannerPolicy, &f.Expires, &f.Version)
	return f, mapped(e)
}
func (s *Store) CreateMedia(ctx context.Context, p d.Principal, id, key string, m d.MediaRequest) (out d.File, e error) {
	if e = m.Validate(); e != nil {
		return
	}
	if !d.Key(key) {
		return out, d.Fail("INVALID_KEY", 400)
	}
	t, e := s.Task(ctx, id)
	if e != nil {
		return
	}
	c := d.Command{Caller: p.Caller(), Action: "CreateMediaSession", OrderID: t.OrderID, Generation: t.Generation, Key: key, Hash: d.Hash(map[string]any{"action": "CreateMediaSession", "order_id": t.OrderID, "task_id": id, "body": m})}
	var fileID string
	e = s.orderTx(ctx, t.OrderID, func(tx pgx.Tx, now time.Time) error {
		current, err := loadTask(ctx, tx, id)
		if err != nil {
			return err
		}
		if !p.Can("FULFILLMENT_MEDIA_UPLOAD", current.Warehouse) {
			return d.Fail("FORBIDDEN", 403)
		}
		if b, found, err := registry(ctx, tx, c); err != nil {
			return err
		} else if found {
			var f struct {
				ID string `json:"file_id"`
			}
			if err = json.Unmarshal(b, &f); err != nil {
				return err
			}
			fileID = f.ID
			return nil
		}
		if current.Assignee == nil || *current.Assignee != p.ID || current.State != "IN_PROGRESS" || m.PackageID != current.Package.ID {
			return d.Fail("TASK_NOT_ASSIGNED", 409)
		}
		if m.Size > current.Inputs.Policy.MaxVideoBytes || current.Inputs.Policy.MaxVideoSeconds <= 0 {
			return d.Fail("MEDIA_POLICY_CONFLICT", 409)
		}
		fileID = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO fulfillment.media_files(file_id,task_id,package_id,purpose,uploader_id,status,object_key,declared_size_bytes,media_type,declared_sha256,scanner_policy_version,session_expires_at,retention_until,version,created_at) VALUES($1,$2,$3,'PACKING_VIDEO',$4,'CREATED',$5,$6,$7,$8,'clamav-ffprobe.v1',$9,$10,1,$11)`, fileID, id, m.PackageID, p.ID, "packing-evidence/staging/"+fileID, m.Size, m.Type, m.Hash, now.Add(10*time.Minute), now.Add(90*24*time.Hour), now)
		if err != nil {
			return err
		}
		if err = audit(ctx, tx, t.OrderID, p.Caller(), c.Action, "", map[string]any{}, map[string]any{"file_id": fileID, "size": m.Size}, now); err != nil {
			return err
		}
		return saveCommand(ctx, tx, c, "APPLIED", "CREATED", "", &id, now, map[string]any{"file_id": fileID})
	})
	if e != nil {
		return
	}
	return s.File(ctx, fileID)
}
func (s *Store) CompleteMedia(ctx context.Context, p d.Principal, id, key string) (out d.File, e error) {
	if !d.Key(key) {
		return out, d.Fail("INVALID_KEY", 400)
	}
	f, e := s.File(ctx, id)
	if e != nil {
		return
	}
	t, e := s.Task(ctx, f.TaskID)
	if e != nil {
		return
	}
	c := d.Command{Caller: p.Caller(), Action: "CompleteMediaUpload", OrderID: t.OrderID, Generation: t.Generation, Key: key, Hash: d.Hash(map[string]any{"action": "CompleteMediaUpload", "order_id": t.OrderID, "file_id": id, "body": map[string]any{}})}
	e = s.orderTx(ctx, t.OrderID, func(tx pgx.Tx, now time.Time) error {
		current, err := loadTask(ctx, tx, t.ID)
		if err != nil {
			return err
		}
		if !p.Can("FULFILLMENT_MEDIA_UPLOAD", current.Warehouse) || p.ID != f.Uploader {
			return d.Fail("FORBIDDEN", 403)
		}
		if b, found, err := registry(ctx, tx, c); err != nil {
			return err
		} else if found {
			return json.Unmarshal(b, &out)
		}
		if current.Assignee == nil || *current.Assignee != p.ID || current.State != "IN_PROGRESS" {
			return d.Fail("TASK_NOT_ASSIGNED", 409)
		}
		var status string
		var expires time.Time
		err = tx.QueryRow(ctx, `SELECT status,session_expires_at FROM fulfillment.media_files WHERE file_id=$1 FOR UPDATE`, id).Scan(&status, &expires)
		if err != nil {
			return err
		}
		if !now.Before(expires) {
			return d.Fail("MEDIA_SESSION_EXPIRED", 410)
		}
		if status != "CREATED" {
			return d.Fail("INVALID_MEDIA_STATE", 409)
		}
		_, err = tx.Exec(ctx, `UPDATE fulfillment.media_files SET status='VERIFYING',version=version+1 WHERE file_id=$1`, id)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO fulfillment.jobs(job_id,kind,dedupe_key,state,stage,order_id,task_id,input,next_run_at,created_at,updated_at) VALUES($1,'MEDIA_VERIFY',$2,'PENDING','VERIFY',$3,$4,$5,$6,$6,$6)`, uuid.NewString(), "media:"+id, t.OrderID, t.ID, marshal(map[string]any{"file_id": id}), now)
		if err != nil {
			return err
		}
		out = f
		out.Status = "VERIFYING"
		return saveCommand(ctx, tx, c, "APPLIED", "VERIFYING", "", &t.ID, now, out)
	})
	return
}
func (s *Store) FinishMedia(ctx context.Context, f d.File, finalKey, version, checksum string, safe bool) error {
	t, e := s.Task(ctx, f.TaskID)
	if e != nil {
		return e
	}
	return s.orderTx(ctx, t.OrderID, func(tx pgx.Tx, now time.Time) error {
		status := "REJECTED"
		scan := "REJECTED"
		if safe {
			status = "VERIFIED"
			scan = "SAFE"
		}
		result, e := tx.Exec(ctx, `UPDATE fulfillment.media_files SET status=$3,scan_status=$4,final_object_key=NULLIF($5,''),object_version=NULLIF($6,''),verified_sha256=NULLIF($7,''),verified_at=CASE WHEN $3='VERIFIED' THEN $8::timestamptz ELSE NULL END,version=version+1 WHERE file_id=$1 AND version=$2 AND status='VERIFYING'`, f.ID, f.Version, status, scan, finalKey, version, checksum, now)
		if e != nil {
			return e
		}
		if result.RowsAffected() != 1 {
			return d.Fail("STALE_MEDIA_RESULT", 409)
		}
		return audit(ctx, tx, t.OrderID, "service:fulfillment-service", "VerifyMedia", "", map[string]any{"file_id": f.ID}, map[string]any{"status": status}, now)
	})
}
func (s *Store) MediaAccessAudit(ctx context.Context, p d.Principal, f d.File, purpose string) (id string, e error) {
	t, e := s.Task(ctx, f.TaskID)
	if e != nil {
		return "", e
	}
	if !p.Can("FULFILLMENT_MEDIA_VIEW", t.Warehouse) {
		return "", d.Fail("FORBIDDEN", 403)
	}
	if purpose != "PACKING_REVIEW" && purpose != "QUALITY_INVESTIGATION" {
		return "", d.Fail("CASE_AUTHORITY_UNAVAILABLE", 503)
	}
	id = uuid.NewString()
	e = s.orderTx(ctx, t.OrderID, func(tx pgx.Tx, now time.Time) error {
		return audit(ctx, tx, t.OrderID, p.Caller(), "MediaAccess", purpose, map[string]any{}, map[string]any{"access_audit_id": id, "file_id": f.ID}, now)
	})
	return
}
