package worker

import "testing"

func TestDecodePackingCompleted(t *testing.T) {
	payload := []byte(`{
		"specversion":"1.0",
		"id":"event-1",
		"type":"vn.omama.fulfillment.packing.completed.v1",
		"data":{
			"order_id":"ord-1",
			"package_id":"pkg-1",
			"seal_code":"seal-1",
			"parcel":{"weight_grams":1200,"length_cm":30,"width_cm":20,"height_cm":15,"fragile":true}
		}
	}`)
	event, err := decodePackingCompleted(payload)
	if err != nil {
		t.Fatal(err)
	}
	if event.OrderID != "ord-1" || event.PackageID != "pkg-1" || event.Parcel.WeightGrams != 1200 || !event.Parcel.Fragile {
		t.Fatalf("unexpected decoded event: %+v", event)
	}
}

func TestDecodePackingCompletedRejectsInvalidParcel(t *testing.T) {
	payload := []byte(`{
		"specversion":"1.0",
		"id":"event-1",
		"type":"vn.omama.fulfillment.packing.completed.v1",
		"data":{"order_id":"ord-1","package_id":"pkg-1","seal_code":"seal-1","parcel":{}}
	}`)
	if _, err := decodePackingCompleted(payload); err == nil {
		t.Fatal("expected invalid parcel to be rejected")
	}
}
