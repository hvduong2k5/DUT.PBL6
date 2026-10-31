// Local test fixture only: invoked by scripts/test-order-inventory.py with a disposable database.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	provider "dut-pbl6/inventory-service/internal/presentation/grpc"
	api "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	"encoding/json"
	"encoding/pem"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	db, err := sql.Open("postgres", os.Getenv("INVENTORY_TEST_DATABASE_URL"))
	must(err)
	db.SetMaxOpenConns(1)
	schema := "peer_" + uuid.NewString()
	schema = "\"" + schema + "\""
	_, err = db.Exec("CREATE SCHEMA " + schema)
	must(err)
	defer func() { db.Exec("DROP SCHEMA " + schema + " CASCADE"); db.Close() }()
	_, err = db.Exec("SET search_path TO " + schema + ", public")
	must(err)
	for _, name := range []string{"000001_init_inventory_schema.up.sql", "000002_terminal_profile.up.sql"} {
		b, e := os.ReadFile(filepath.Join("migrations", name))
		must(e)
		_, e = db.Exec(string(b))
		must(e)
	}
	_, err = db.Exec(`INSERT INTO inventory_items(sku,warehouse_id,physical_qty) VALUES('PEER','00000000-0000-0000-0000-000000000001',100)`)
	must(err)
	_, err = db.Exec(`INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,status) VALUES($1,'PEER','PEER',$2,CURRENT_DATE-1,CURRENT_DATE+30,100,'ACTIVE')`, uuid.NewString(), uuid.NewString())
	must(err)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(err)
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	must(err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	must(os.WriteFile(os.Getenv("PEER_CA_FILE"), certPEM, 0600))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	must(err)
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12})))
	api.RegisterInventoryServiceServer(server, &provider.TerminalServer{DB: db, Token: os.Getenv("PEER_TOKEN")})
	go func() { must(server.Serve(listener)) }()
	must(os.WriteFile(os.Getenv("PEER_READY_FILE"), func() []byte { b, _ := json.Marshal(map[string]string{"address": listener.Addr().String()}); return b }(), 0600))
	for {
		if _, err := os.Stat(os.Getenv("PEER_STOP_FILE")); err == nil {
			server.Stop()
			var physical, reserved, batchPhysical, batchReserved, events int
			must(db.QueryRow(`SELECT physical_qty,reserved_qty FROM inventory_items WHERE sku='PEER'`).Scan(&physical, &reserved))
			must(db.QueryRow(`SELECT physical_qty,reserved_qty FROM batches WHERE sku='PEER'`).Scan(&batchPhysical, &batchReserved))
			must(db.QueryRow(`SELECT count(*) FROM outbox_events WHERE event_type='vn.omama.inventory.stock.committed.v1'`).Scan(&events))
			summary, err := json.Marshal(map[string]int{"physical": physical, "reserved": reserved, "batch_physical": batchPhysical, "batch_reserved": batchReserved, "events": events})
			must(err)
			must(os.WriteFile(os.Getenv("PEER_RESULT_FILE"), summary, 0600))
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
