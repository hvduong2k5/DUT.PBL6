package adapters

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/omamx/fulfillment-service/internal/application"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestClamdProtocolVerdicts(t *testing.T) {
	for _, verdict := range []string{"stream: OK", "stream: Eicar-Test-Signature FOUND", "stream: daemon ERROR"} {
		t.Run(verdict, func(t *testing.T) {
			listener, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			defer listener.Close()
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, e := listener.Accept()
				if e != nil {
					return
				}
				defer conn.Close()
				header := make([]byte, 10)
				if _, e = io.ReadFull(conn, header); e != nil {
					return
				}
				for {
					var size [4]byte
					if _, e = io.ReadFull(conn, size[:]); e != nil {
						return
					}
					n := binary.BigEndian.Uint32(size[:])
					if n == 0 {
						break
					}
					if _, e = io.CopyN(io.Discard, conn, int64(n)); e != nil {
						return
					}
				}
				_, _ = conn.Write([]byte(verdict + "\x00"))
			}()
			path := filepath.Join(t.TempDir(), "synthetic.bin")
			if e = os.WriteFile(path, []byte("harmless protocol fixture"), 0600); e != nil {
				t.Fatal(e)
			}
			e = (Clamd{Address: listener.Addr().String()}).Scan(context.Background(), path)
			<-done
			if verdict == "stream: OK" && e != nil {
				t.Fatal(e)
			}
			if verdict == "stream: Eicar-Test-Signature FOUND" && !errors.Is(e, application.ErrUnsafeMedia) {
				t.Fatal("malware verdict not classified", e)
			}
			if verdict == "stream: daemon ERROR" && (e == nil || errors.Is(e, application.ErrUnsafeMedia)) {
				t.Fatal("scanner outage treated as a clean or malware verdict", e)
			}
		})
	}
}
