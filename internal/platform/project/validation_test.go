package project

import "testing"

func TestValidProjectIdentity(t *testing.T) {
	if !validProjectIdentity("支付平台", "payment-platform") {
		t.Fatal("expected a valid project identity")
	}
	if validProjectIdentity("支付平台", "Payment") || validProjectIdentity("", "payment") {
		t.Fatal("invalid project identity was accepted")
	}
}
