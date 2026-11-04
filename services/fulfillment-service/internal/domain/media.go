package domain

import "time"

type File struct {
	ID                                                                         string     `json:"file_id"`
	TaskID                                                                     string     `json:"task_id"`
	PackageID                                                                  string     `json:"package_id"`
	Status                                                                     string     `json:"status"`
	Type                                                                       string     `json:"media_type"`
	Size                                                                       int64      `json:"size_bytes"`
	Checksum                                                                   *string    `json:"verified_sha256"`
	VerifiedAt                                                                 *time.Time `json:"verified_at"`
	Uploader, StagingKey, FinalKey, ObjectVersion, DeclaredHash, ScannerPolicy string     `json:"-"`
	Expires                                                                    time.Time  `json:"-"`
	Version                                                                    int64      `json:"-"`
}
type MediaRequest struct {
	PackageID string `json:"package_id"`
	Type      string `json:"media_type"`
	Size      int64  `json:"size_bytes"`
	Hash      string `json:"sha256"`
}

func (m MediaRequest) Validate() error {
	if !UUID(m.PackageID) || !contains([]string{"video/mp4", "video/webm"}, m.Type) || m.Size < 1 || m.Size > 209715200 || !hashPattern.MatchString(m.Hash) {
		return Fail("INVALID_MEDIA", 400)
	}
	return nil
}
