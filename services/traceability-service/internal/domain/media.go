package domain

import (
	"sort"
	"strings"
)

// CertificateMediaIDs maps immutable evidence references to the existing Media file IDs.
// It does not change stored/public content or expose internal references.
func CertificateMediaIDs(content Content) ([]string, error) {
	evidence := map[string]string{}
	for _, item := range content.Evidence {
		evidence[strings.ToLower(item.ID)] = item.Reference
	}
	unique := map[string]bool{}
	for _, cert := range content.Certificates {
		if cert.Type != "OCOP" {
			continue
		}
		for _, id := range cert.EvidenceIDs {
			reference := evidence[strings.ToLower(id)]
			fileID, ok := strings.CutPrefix(reference, "document:")
			if !ok || !ValidUUID(fileID) || fileID != strings.ToLower(fileID) {
				return nil, invalid("certificate.media_reference")
			}
			unique[fileID] = true
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}
