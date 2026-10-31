package http

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
)

type checkoutWard struct {
	Code string `json:"wardCode"`
	Name string `json:"wardName"`
}
type checkoutProvince struct {
	Code  string         `json:"provinceCode"`
	Name  string         `json:"provinceName"`
	Wards []checkoutWard `json:"wards"`
}

// Read the same canonical parent/child data used to validate address writes.
// Deployment owns importing and reviewing the supported administrative dataset.
func locationsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			writeJSONError(w, 503, "locations unavailable")
			return
		}
		rows, err := pool.Query(r.Context(), `SELECT p.code,p.name,w.code,w.name FROM administrative_units p JOIN administrative_units w ON w.parent_code=p.code AND w.level='WARD' WHERE p.level='PROVINCE' ORDER BY p.code,w.code`)
		if err != nil {
			writeJSONError(w, 503, "locations unavailable")
			return
		}
		defer rows.Close()
		provinces := []checkoutProvince{}
		for rows.Next() {
			var pc, pn, wc, wn string
			if err = rows.Scan(&pc, &pn, &wc, &wn); err != nil {
				writeJSONError(w, 503, "locations unavailable")
				return
			}
			if len(provinces) == 0 || provinces[len(provinces)-1].Code != pc {
				provinces = append(provinces, checkoutProvince{Code: pc, Name: pn, Wards: []checkoutWard{}})
			}
			i := len(provinces) - 1
			provinces[i].Wards = append(provinces[i].Wards, checkoutWard{Code: wc, Name: wn})
		}
		if rows.Err() != nil || len(provinces) == 0 {
			writeJSONError(w, 503, "locations unavailable")
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=300")
		writeJSON(w, 200, map[string]any{"provinces": provinces})
	}
}
