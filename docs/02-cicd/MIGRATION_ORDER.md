# CI/CD Migration Order

The complete baseline is `001` through `005`, followed by CI/CD migrations:

```text
001_platform_base.sql
002_asset_management.sql
003_cicd_core.sql
004_authorization_scope.sql
005_identity_link.sql
006_cicd_build_execution.sql
007_cicd_project_delivery.sql
008_cicd_service_revision.sql
009_cicd_project_environment.sql
010_cicd_environment_service_override.sql
011_cicd_workflow.sql
012_cicd_permission_defaults.sql
013_cicd_workflow_run.sql
014_cicd_workflow_governance.sql
015_cicd_workflow_task_lease.sql
016_cicd_deployment_snapshot.sql
017_cicd_deployment_rollback.sql
018_cicd_build_observability.sql
019_cicd_code_source_permissions.sql
020_cicd_webhook_delivery.sql
021_cicd_notification_outbox.sql
022_cicd_host_agent.sql
023_cicd_host_artifact_integrity.sql
024_cicd_host_rolling_batch.sql
025_cicd_helm_execution.sql
026_cicd_build_governance.sql
027_cicd_build_output_verification.sql
028_access_department_workflow_scope.sql
029_cicd_project_initialization.sql
030_cicd_environment_resources.sql
031_cicd_build_fingerprint_and_mode.sql
032_cicd_build_cache_strategy.sql
033_cicd_platform_build_settings.sql
034_system_build_resources_menu.sql
035_cicd_registry_namespaces.sql
036_build_execution_reliability.sql
037_build_job_retention_default.sql
038_workflow_task_log.sql
039_delivery_templates.sql
040_workflow_environment_binding.sql
041_workflow_task_retry.sql
042_backfill_workflow_environment_bindings.sql
043_build_git_changes.sql
044_workflow_run_numbers.sql
045_inline_dockerfile.sql
046_environment_yaml_revisions.sql
047_cicd_default_builder_command.sql
048_project_code_reuse.sql
049_verified_production_release.sql
050_production_environment_classification.sql
051_observability_resource_links.sql
052_delivery_template_versions.sql
```

Do not execute an individual CI/CD file out of order. The migration runner
locks the database, checks checksums and records each version in
`schema_migration`.

```powershell
# Apply pending migrations
go run ./cmd/axiops-migrate -f configs/platform-api.yaml

# Verify without applying DDL
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

Migration 052 adds immutable delivery-template history. Apply migrations in order
before activating the matching CI/CD API, Worker and frontend. Current template
contents are backfilled at startup; overwritten pre-migration history cannot be
reconstructed. The migration does not change environment bindings or deploy.

No business database migrations were executed during this implementation turn.
Integration tests apply schemas only in disposable test databases.
