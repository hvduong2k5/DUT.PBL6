package sepay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestExactWebhookProtocol(t *testing.T) {
	raw := []byte(`{"id":9007199254740993,"gateway":"VCB","transactionDate":"2026-10-24 12:00:00","accountNumber":"00001","content":"OMP7K9M2N4P6R8T","transferType":"in","transferAmount":1000000000001}`)
	now := time.Now().Truncate(time.Second)
	stamp := strconv.FormatInt(now.Unix(), 10)
	secret := strings.Repeat("s", 32)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stamp + "."))
	h.Write(raw)
	signature := "sha256=" + hex.EncodeToString(h.Sum(nil))
	if Verify(raw, stamp, signature, secret, now) != nil {
		t.Fatal("valid raw HMAC rejected")
	}
	w, e := ParseWebhook(raw)
	if e != nil || w.ID != 9007199254740993 || w.Amount != 1000000000001 {
		t.Fatal("lossy int64", w, e)
	}
	for _, tc := range []struct {
		name             string
		raw              []byte
		stamp, signature string
		at               time.Time
	}{{"tamper", append(raw, ' '), stamp, signature, now}, {"expired", raw, stamp, signature, now.Add(301 * time.Second)}, {"future", raw, stamp, signature, now.Add(-301 * time.Second)}, {"missing", raw, stamp, "", now}} {
		t.Run(tc.name, func(t *testing.T) {
			if Verify(tc.raw, tc.stamp, tc.signature, secret, tc.at) == nil {
				t.Fatal("accepted invalid HMAC")
			}
		})
	}
	for _, s := range []string{`{"id":1,"id":2}`, `{"id":1,"extra":{"x":1,"x":2}}`, `{} {}`, `{"id":1.5}`, strings.Replace(string(raw), "9007199254740993", "9223372036854775808", 1)} {
		if _, e := ParseWebhook([]byte(s)); e == nil {
			t.Fatal("accepted invalid webhook", s)
		}
	}
}
func TestReferenceParser(t *testing.T) {
	for _, tc := range []struct{ content, hint, code, reason string }{
		{"payment omp7k9m2n4p6r8t done", "", "OMP7K9M2N4P6R8T", ""},
		{"XOMP7K9M2N4P6R8T", "", "", "UNMATCHED_REFERENCE"},
		{"OMP7K9M2N4P6R8TX", "", "", "UNMATCHED_REFERENCE"},
		{"OMP7K9M2N4P6R8T OMP7K9M2N4P6R8T", "", "", "AMBIGUOUS_REFERENCE"},
		{"OMP7K9M2N4P6R8T", "OMP000000000000", "", "REFERENCE_CONFLICT"},
		{"no code", "OMP7K9M2N4P6R8T", "", "UNMATCHED_REFERENCE"},
	} {
		code, reason := PaymentCode(tc.content, tc.hint)
		if code != tc.code || reason != tc.reason {
			t.Fatal(tc, code, reason)
		}
	}
}
func TestHTTPReaderAndFailureContract(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"empty page", `{"status":"success","data":[],"meta":{"pagination":{"current_page":1,"per_page":100,"last_page":1,"total":0,"has_more":false}}}`, 200, false},
		{"pagination absent", `{"status":"success","data":[]}`, 200, true},
		{"duplicate envelope", `{"status":"success","status":"success","data":[]}`, 200, true},
		{"upstream HTML", "<html>failure</html>", 500, true}, {"throttle", "{}", 429, true}, {"auth", "{}", 401, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer synthetic" {
					t.Error("bearer absent")
				}
				if r.URL.Query().Get("bank_account_id") != "bank" || r.URL.Query().Get("since_id") != "" {
					t.Error("wrong window/filter")
				}
				w.Header().Set("Retry-After", "2")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			client := Client{base: server.URL, token: "synthetic", http: server.Client()}
			_, e := client.List(context.Background(), Window{Bank: "bank", From: time.Now().Add(-time.Hour), To: time.Now(), Page: 1})
			if (e != nil) != tc.wantError {
				t.Fatal(e)
			}
		})
	}
}
func TestQRAndFinancialIdentity(t *testing.T) {
	receiver := Receiver{BankID: "0756c5ee-c9d0-11f1-b21a-a6006ab65aca", BankCode: "VCB", BankName: "Vietcombank", Account: "00000001", Holder: "SYNTHETIC", Mode: "BANK_ACCOUNT", Currency: "VND"}
	url, e := receiver.QR("OMP7K9M2N4P6R8T", 10000)
	if e != nil || !strings.Contains(url, "acc=00000001") || !strings.Contains(url, "amount=10000") {
		t.Fatal(url, e)
	}
	receiver.Mode = "CONTENT_VA"
	if _, e = receiver.QR("OMP7K9M2N4P6R8T", 10000); e == nil {
		t.Fatal("unsupported VA auto-enabled")
	}
	a := Transaction{ID: "ed47ed5c-cf6e-11f1-b21a-a6006ab65aca", BankAccountID: "0756c5ee-c9d0-11f1-b21a-a6006ab65aca", Account: "00000001", Direction: "in", In: 10000, Date: "2026-10-24T12:00:00+07:00", Content: "OMP7K9M2N4P6R8T"}
	b := a
	b.Code = "derived"
	first, _ := a.FinancialBytes()
	second, _ := b.FinancialBytes()
	if string(first) != string(second) {
		t.Fatal("derived code changed financial hash")
	}
	b.In++
	second, _ = b.FinancialBytes()
	if string(first) == string(second) {
		t.Fatal("amount absent from financial hash")
	}
}
