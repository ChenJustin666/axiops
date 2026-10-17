package helmconfig

import "testing"

func TestNativeValuesKeepTypesAndRejectAmbiguousYAML(t *testing.T) {
	v, err := ParseValues("replicas: 2\nenabled: true\nenv:\n- name: MODE\n  value: test\nresources:\n  memory: 2Gi\n")
	if err != nil || v["replicas"] != 2 || v["enabled"] != true || len(v["env"].([]interface{})) != 1 {
		t.Fatalf("types lost: %#v %v", v, err)
	}
	for _, text := range []string{"a: 1\na: 2", "---\na: 1\n---\nb: 2", "- a\n- b", "str", "parent:\n  ? [a, b]\n  : c"} {
		if _, err := ParseValues(text); err == nil {
			t.Errorf("accepted invalid values: %q", text)
		}
	}
	for _, text := range []string{"", "{}", "a: null", "text: 'true'"} {
		if _, err := ParseValues(text); err != nil {
			t.Errorf("valid values rejected: %q %v", text, err)
		}
	}
}

func TestImageBindingRequiresOneUnambiguousMapping(t *testing.T) {
	for _, binding := range []ImageBinding{{RepositoryPath: "image", TagPath: "image.tag"}, {RepositoryPath: "a", TagPath: "a"}, {RefPath: "image", TagPath: "image.tag"}, {RefPath: "../image"}, {}} {
		s := Default()
		s.Image = binding
		if s.Validate() == nil {
			t.Errorf("accepted ambiguous mapping: %+v", binding)
		}
	}
	s := Default()
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Image = ImageBinding{RefPath: "gateway.image"}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
