package domain

import "time"

// ExpiryTransition identifies the first due certificate in immutable order.
// The same revision/reason/cutoff produces exactly one local observation.
func (c Content) ExpiryTransition(now time.Time) (string, time.Time, bool) {
	for _, cert := range c.Certificates {
		if cert.NoExpiryStated {
			if cert.ReviewDueAt != nil && !now.Before(*cert.ReviewDueAt) {
				return "CERTIFICATE_REVIEW_DUE", cert.ReviewDueAt.UTC(), true
			}
		} else if cert.ValidUntil != nil {
			end, err := certificateDate(*cert.ValidUntil)
			if err != nil {
				continue
			}
			cutoff := end.AddDate(0, 0, 1)
			if !now.Before(cutoff) {
				return "CERTIFICATE_EXPIRED", cutoff.UTC(), true
			}
		}
	}
	return "", time.Time{}, false
}
