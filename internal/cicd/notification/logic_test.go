package notification

import (
	"context"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type notificationPolicyStore []authz.Policy

func (p notificationPolicyStore) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) {
	return p, nil
}

type fakeLogicRepository struct {
	deliveriesCalled bool
	projectID        int64
	status           string
	page             int
	pageSize         int
}

func (f *fakeLogicRepository) Create(context.Context, Subscription) (int64, error) {
	return 1, nil
}
func (f *fakeLogicRepository) List(context.Context, int64, authz.ScopeSet) ([]Subscription, error) {
	return nil, nil
}
func (f *fakeLogicRepository) Delete(context.Context, int64, int64) error { return nil }
func (f *fakeLogicRepository) FindCredential(context.Context, int64) (*Credential, error) {
	return nil, nil
}
func (f *fakeLogicRepository) ListDeliveries(_ context.Context, projectID int64, status string, page, pageSize int, _ authz.ScopeSet) ([]Delivery, int64, error) {
	f.deliveriesCalled = true
	f.projectID, f.status, f.page, f.pageSize = projectID, status, page, pageSize
	return []Delivery{{ID: 9, ProjectID: projectID, Status: status}}, 1, nil
}

func TestListDeliveriesUsesNotificationViewScopeAndNormalizesPage(t *testing.T) {
	repo := &fakeLogicRepository{}
	logic := NewLogic(nil, authz.New(notificationPolicyStore{{
		Resource: "notification", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}}), "test-key")
	logic.repo = repo
	items, total, err := logic.ListDeliveries(context.Background(), 7, 10, "failed", 0, 500)
	if err != ErrInvalid || repo.deliveriesCalled || items != nil || total != 0 {
		t.Fatal("invalid delivery status was accepted")
	}
	items, total, err = logic.ListDeliveries(context.Background(), 7, 10, "dead", 0, 500)
	if err != nil || total != 1 || len(items) != 1 || !repo.deliveriesCalled {
		t.Fatalf("list deliveries failed: total=%d items=%v err=%v", total, items, err)
	}
	if repo.projectID != 10 || repo.status != "dead" || repo.page != 1 || repo.pageSize != 100 {
		t.Fatalf("unexpected normalized query: %+v", repo)
	}
}

func TestListDeliveriesRequiresViewPermission(t *testing.T) {
	repo := &fakeLogicRepository{}
	logic := NewLogic(nil, authz.New(notificationPolicyStore{}), "test-key")
	logic.repo = repo
	_, _, err := logic.ListDeliveries(context.Background(), 7, 10, "", 1, 20)
	if err != ErrNoPermission || repo.deliveriesCalled {
		t.Fatal("delivery records were exposed without notification:view")
	}
}
