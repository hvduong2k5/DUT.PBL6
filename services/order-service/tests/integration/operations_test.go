package integration

import (
	"context"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	order "github.com/omamx/order-service/internal/gen/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"sync"
	"testing"
)

func TestV11ReadyCancelBarrierAndStaleAuthorization(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	_, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "paid cancellation", false)
	requireCode(t, e, "CANCELLATION_NOT_ALLOWED")
	cod := h.place("COD")
	h.jobs()
	cod = h.order(cod.ID)
	_, e = h.S.Cancel(h.ctx, h.Principal, cod.ID, uuid.NewString(), cod.Version, "test", false)
	if e != nil {
		t.Fatal(e)
	}
	h.jobs()
	var result struct {
		Authorized bool `json:"authorized"`
	}
	if code := h.request(h.Mock.URL, "POST", "/fulfillment/authorize", "", map[string]any{"order_id": cod.ID, "generation": cod.Generation}, &result); code != 200 || result.Authorized {
		t.Fatal("stale ready resurrected cancelled task")
	}
	if h.request(h.Mock.URL, "POST", "/fulfillment/accept", "", map[string]any{"order_id": cod.ID, "generation": cod.Generation}, nil) != 409 {
		t.Fatal("cancelled task accepted")
	}
}
func TestV11InboxEffectRollbackAndReplay(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	h.acceptTask(o)
	event := h.source(o, "PACKING_ACCEPTED")
	h.exec(`CREATE FUNCTION fail_inbox_checkpoint() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected inbox checkpoint failure'; END $$`)
	h.exec(`CREATE TRIGGER fail_inbox_checkpoint BEFORE UPDATE ON inbox FOR EACH ROW EXECUTE FUNCTION fail_inbox_checkpoint()`)
	if _, e := h.S.ProcessEvent(h.ctx, event); e == nil {
		t.Fatal("injected checkpoint failure did not abort transaction")
	}
	if current := h.order(o.ID); current.Status != o.Status || current.Version != o.Version {
		t.Fatal("Order effect survived inbox transaction rollback")
	}
	if h.count("SELECT count(*) FROM inbox") != 0 || h.count("SELECT count(*) FROM source_projections") != 0 {
		t.Fatal("inbox or source checkpoint survived rollback")
	}
	h.exec("DROP TRIGGER fail_inbox_checkpoint ON inbox")
	if h.apply(event) != "APPLIED" || h.order(o.ID).Status != "PROCESSING" {
		t.Fatal("replayed event did not apply")
	}
	version := h.order(o.ID).Version
	if h.apply(event) != "APPLIED" || h.order(o.ID).Version != version || h.count("SELECT count(*) FROM order_history WHERE to_status='PROCESSING'") != 1 {
		t.Fatal("duplicate replay produced a second business effect")
	}
}
func TestV11CancelVersusPackingAcceptance(t *testing.T) {
	h := newHarness(t)
	o := h.place("COD")
	h.jobs()
	o = h.order(o.ID)
	_, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "race", false)
	if e != nil {
		t.Fatal(e)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var code int
	var jobErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		code = h.request(h.Mock.URL, "POST", "/fulfillment/accept", "", map[string]any{"order_id": o.ID, "generation": o.Generation}, nil)
	}()
	go func() { defer wg.Done(); <-start; _, jobErr = h.S.WorkJob(h.ctx) }()
	close(start)
	wg.Wait()
	if jobErr != nil {
		t.Fatal(jobErr)
	}
	h.jobs()
	current := h.order(o.ID)
	if code == 200 {
		if current.Status == "CANCELLED_BY_USER" || current.Stock != "COMMITTED" {
			t.Fatal("task accepted but stock released")
		}
		event := h.source(o, "PACKING_ACCEPTED")
		if h.apply(event) != "APPLIED" || h.order(o.ID).Status != "PROCESSING" {
			t.Fatal("accepted task not reconciled")
		}
	} else if code == 409 {
		if current.Status != "CANCELLED_BY_USER" || current.Stock != "RELEASED" {
			t.Fatalf("cancel won incorrectly %+v", current)
		}
	} else {
		t.Fatalf("unexpected task status %d", code)
	}
}
func TestV11OutOfOrderPackingAndShipping(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	h.acceptTask(o)
	sealed := h.source(o, "PACKING_SEALED")
	if h.apply(sealed) != "DEFERRED" {
		t.Fatal("sealed applied before acceptance")
	}
	accepted := h.source(o, "PACKING_ACCEPTED")
	if h.apply(accepted) != "APPLIED" {
		t.Fatal("acceptance not applied")
	}
	if e := h.S.ReplayDeferred(h.ctx); e != nil {
		t.Fatal(e)
	}
	h.exec("UPDATE inbox SET run_after=clock_timestamp() WHERE status='DEFERRED'")
	if e := h.S.ReplayDeferred(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Status != "PACKED" {
		t.Fatal("sealed event lost")
	}
	delivered := h.source(o, "SHIPMENT_DELIVERED")
	if h.apply(delivered) != "DEFERRED" {
		t.Fatal("delivered before dispatch")
	}
	created := h.source(o, "SHIPMENT_CREATED")
	h.apply(created)
	if h.order(o.ID).Status != "PACKED" {
		t.Fatal("shipment created changed lifecycle")
	}
	dispatched := h.source(o, "SHIPMENT_DISPATCHED")
	h.apply(dispatched)
	h.exec("UPDATE inbox SET run_after=clock_timestamp() WHERE status='DEFERRED'")
	if e := h.S.ReplayDeferred(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.order(o.ID).Status != "DELIVERED" {
		t.Fatal("delivery event lost")
	}
	before := h.order(o.ID).Version
	delivered.ID = uuid.NewString()
	delivered.Signature = domain.EventSignature(delivered, internalToken)
	h.apply(delivered)
	h.apply(created)
	if h.order(o.ID).Version != before {
		t.Fatal("business checkpoint duplicated with new event ID")
	}
	wrong := accepted
	wrong.ID = uuid.NewString()
	wrong.OrderID = uuid.NewString()
	wrong.Signature = domain.EventSignature(wrong, internalToken)
	_, e := h.S.ProcessEvent(h.ctx, wrong)
	requireCode(t, e, "RESOURCE_MISMATCH")
}
func TestV11StaffScopeQueueAndHold(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	packing := domain.Principal{ID: uuid.NewString(), Role: "PACKING_STAFF", Warehouse: "HUE"}
	rows, e := h.S.ListOrders(h.ctx, packing, true, true, "", "", 20)
	if e != nil || len(rows) != 1 {
		t.Fatalf("queue %v %d", e, len(rows))
	}
	packing.Warehouse = "OTHER"
	rows, e = h.S.ListOrders(h.ctx, packing, true, true, "", "", 20)
	if e != nil || len(rows) != 0 {
		t.Fatal("warehouse scope leaked")
	}
	_, e = h.S.ListOrders(h.ctx, h.Principal, true, true, "", "", 20)
	requireCode(t, e, "PERMISSION_DENIED")
	manager := domain.Principal{ID: uuid.NewString(), Role: "SALES_MANAGER", Warehouse: "HUE"}
	if e = h.S.Hold(h.ctx, manager, o.ID, o.Version); e != nil {
		t.Fatal(e)
	}
	rows, e = h.S.ListOrders(h.ctx, manager, true, true, "", "", 20)
	if e != nil || len(rows) != 0 {
		t.Fatal("held order remains actionable")
	}
}
func TestV11GRPCRegisteredServerAndOwnership(t *testing.T) {
	h := newHarness(t)
	o := h.place("VIETQR")
	conn, e := grpc.DialContext(h.ctx, h.GRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	client := order.NewOrderServiceClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+h.Token))
	response, e := client.GetOrderDetail(ctx, &order.GetOrderDetailRequest{OrderId: o.ID, RequestingUserId: uuid.NewString()})
	if e != nil || response.Version <= 0 || response.CustomerId == h.Principal.ID || len(response.Items) != 1 {
		t.Fatalf("registered gRPC contract %v", e)
	}
	other := domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER"}
	otherToken := h.token(other)
	ctx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+otherToken))
	if _, e = client.GetOrderDetail(ctx, &order.GetOrderDetailRequest{OrderId: o.ID, RequestingUserId: h.Principal.ID}); e == nil {
		t.Fatal("requesting_user_id bypassed authenticated owner")
	}
}

func TestV11HeldOrderCannotStartPacking(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	manager := domain.Principal{ID: uuid.NewString(), Role: "SALES_MANAGER", Warehouse: "HUE"}
	if e := h.S.Hold(h.ctx, manager, o.ID, o.Version); e != nil {
		t.Fatal(e)
	}
	if h.request(h.Mock.URL, "POST", "/fulfillment/accept", "", map[string]any{"order_id": o.ID, "generation": o.Generation}, nil) != 409 {
		t.Fatal("cached ready authorization bypassed current hold")
	}
	if h.mockCount("SELECT count(*) FROM fulfillment WHERE state='ACCEPTED'") != 0 {
		t.Fatal("held order started packing")
	}
}

func TestV11AcceptedTaskRejectsLaterCancellation(t *testing.T) {
	h := newHarness(t)
	o := h.place("COD")
	h.jobs()
	o = h.order(o.ID)
	h.acceptTask(o)
	if _, e := h.S.Cancel(h.ctx, h.Principal, o.ID, uuid.NewString(), o.Version, "late request", false); e != nil {
		t.Fatal(e)
	}
	h.jobs()
	if h.order(o.ID).Stock != "COMMITTED" || h.order(o.ID).Status == "CANCELLED_BY_USER" {
		t.Fatal("accepted task stock was reversed")
	}
	h.apply(h.source(o, "PACKING_ACCEPTED"))
	if h.order(o.ID).Status != "PROCESSING" {
		t.Fatal("acceptance checkpoint was lost")
	}
}

func TestV11UnsignedSourceCannotForgeDelivery(t *testing.T) {
	h := newHarness(t)
	o := h.paid()
	h.acceptTask(o)
	event := h.source(o, "PACKING_ACCEPTED")
	event.Type = "SHIPMENT_DELIVERED"
	_, e := h.S.ProcessEvent(h.ctx, event)
	requireCode(t, e, "INVALID_EVENT_SIGNATURE")
	if h.order(o.ID).Status != "PAID" {
		t.Fatal("forged source changed lifecycle")
	}
}
