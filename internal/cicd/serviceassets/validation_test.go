package serviceassets

import "testing"

func TestRouteAndStorageValidation(t *testing.T) {
	for _, host := range []string{"api.example.com", "*.example.com", "internal"} {
		s := Empty()
		s.Routes = []Route{{Name: "route", Host: host, Path: "/", Service: "app", Port: 8080}}
		if err := Validate(s); err != nil {
			t.Fatalf("valid %s: %v", host, err)
		}
	}
	for _, path := range []string{"/a/../b", "/a//b", "/a?b", "/a b", "/a\\b"} {
		s := Empty()
		s.Routes = []Route{{Name: "route", Host: "api.example.com", Path: path, Service: "app", Port: 80}}
		if Validate(s) == nil {
			t.Fatalf("invalid path accepted: %s", path)
		}
	}
	for _, capacity := range []string{"2", "2gi", "0Gi", "1Ki", "-1Gi"} {
		s := Empty()
		s.Volumes = []Volume{{Name: "data", Capacity: capacity, MountPath: "/data", AccessMode: "ReadWriteOnce"}}
		if Validate(s) == nil {
			t.Fatalf("invalid capacity accepted: %s", capacity)
		}
	}
}
