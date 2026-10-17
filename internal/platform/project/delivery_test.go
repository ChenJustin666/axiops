package project

import "testing"

func TestIsValidDeliveryType(t *testing.T) {
	for _, deliveryType := range []string{DeliveryTypeK8sYAML, DeliveryTypeHelm, DeliveryTypeHost} {
		if !IsValidDeliveryType(deliveryType) {
			t.Fatalf("expected %q to be valid", deliveryType)
		}
	}
	for _, deliveryType := range []string{"", DeliveryTypeUnclassified, "vm", "k8s"} {
		if IsValidDeliveryType(deliveryType) {
			t.Fatalf("expected %q to be invalid", deliveryType)
		}
	}
}

func TestCanConfirmDeliveryType(t *testing.T) {
	if !CanConfirmDeliveryType(DeliveryTypeUnclassified, DeliveryStatusPendingConfirmation, DeliveryTypeHelm) {
		t.Fatal("expected an unclassified pending project to be confirmable")
	}
	cases := []struct {
		currentType   string
		currentStatus string
		nextType      string
	}{
		{DeliveryTypeHelm, DeliveryStatusActive, DeliveryTypeHost},
		{DeliveryTypeUnclassified, DeliveryStatusActive, DeliveryTypeHost},
		{DeliveryTypeUnclassified, DeliveryStatusPendingConfirmation, DeliveryTypeUnclassified},
	}
	for _, tc := range cases {
		if CanConfirmDeliveryType(tc.currentType, tc.currentStatus, tc.nextType) {
			t.Fatalf("unexpected confirmation allowed: %+v", tc)
		}
	}
}
