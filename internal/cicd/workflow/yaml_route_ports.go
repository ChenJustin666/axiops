package workflow

import "github.com/ChenJustin666/axiops/internal/cicd/serviceassets"

// ReconcileYAMLRoutePorts is used only while saving environment configuration.
// Execution continues to validate the immutable snapshot without repairing it.
func ReconcileYAMLRoutePorts(before, after *YAMLSource) (serviceassets.Settings, []string, error) {
	if before == nil || len(before.Assets.Routes) == 0 || len(after.Assets.Routes) == 0 {
		return after.Assets, nil, nil
	}
	oldManifest, err := renderYAMLDefinition(before.Manifest, before, "axiops-config-validation.invalid/pending-image:validation")
	if err != nil {
		return after.Assets, nil, nil
	} // An invalid old draft must remain repairable.
	newManifest, err := renderYAMLDefinition(after.Manifest, after, "axiops-config-validation.invalid/pending-image:validation")
	if err != nil {
		return after.Assets, nil, err
	}
	oldItems, err := ParseYAMLManifest(oldManifest)
	if err != nil {
		return after.Assets, nil, nil
	}
	newItems, err := ParseYAMLManifest(newManifest)
	if err != nil {
		return after.Assets, nil, err
	}
	settings, notices := serviceassets.ReconcileRoutePorts(oldItems, newItems, before.Assets, after.Assets)
	return settings, notices, nil
}
