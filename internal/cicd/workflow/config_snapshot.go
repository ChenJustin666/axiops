package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Resolve bindings and freeze inputs before validation and queuing. This run's
// immutable build output is injected later. Credential payloads are never saved.
func (r *Repository) FreezeDeploymentConfigs(ctx context.Context, environmentID int64, nodes []Node, explicitRevision bool) ([]Node, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result := append([]Node(nil), nodes...)
	for i := range result {
		node := &result[i]
		if node.Type != NodeYAMLDeploy && node.Type != NodeHelmDeploy && node.Type != NodeHostDeploy {
			continue
		}
		config := map[string]json.RawMessage{}
		if err = json.Unmarshal(node.Config, &config); err != nil {
			return nil, err
		}
		var sid, rid int64
		_ = json.Unmarshal(config["service_id"], &sid)
		_ = json.Unmarshal(config["service_revision_id"], &rid)
		if !explicitRevision {
			var bound int64
			err = tx.QueryRowContext(ctx, `SELECT COALESCE(ov.service_revision_id,svc.current_revision_id) FROM app_service svc
   LEFT JOIN environment_service_override ov ON ov.service_id=svc.id AND ov.project_environment_id=? WHERE svc.id=? AND svc.status='active'`, environmentID, sid).Scan(&bound)
			if err != nil {
				return nil, err
			}
			if bound > 0 {
				rid = bound
			}
		}
		if rid <= 0 {
			return nil, fmt.Errorf("service %d has no deployment revision", sid)
		}
		config["service_revision_id"], _ = json.Marshal(rid)
		delete(config, "frozen_yaml")
		delete(config, "frozen_helm")
		delete(config, "frozen_helm_variables")
		delete(config, "frozen_host")
		delete(config, "frozen_agent_ids")
		// Image reuse is injected by the server after successful-build ownership
		// checks, never accepted from a saved/editor-supplied deploy node.
		if node.Type == NodeHelmDeploy {
			delete(config, "image_ref")
		}
		if node.Type == NodeYAMLDeploy {
			source, e := (&SQLYAMLSourceStore{db: tx}).LoadYAMLSource(ctx, environmentID, sid, rid)
			if e != nil {
				return nil, e
			}
			if source == nil {
				return nil, fmt.Errorf("service revision unavailable")
			}
			config["frozen_yaml"], err = json.Marshal(source)
		} else if node.Type == NodeHostDeploy {
			store := &SQLHostSourceStore{db: tx}
			source, e := store.LoadHostSource(ctx, environmentID, sid, rid)
			if e != nil {
				return nil, e
			}
			if source == nil {
				return nil, fmt.Errorf("host revision unavailable")
			}
			var spec HostSpec
			if json.Unmarshal(source.Definition, &spec) != nil || (spec.OS != "linux" && spec.OS != "windows") {
				return nil, fmt.Errorf("invalid host definition")
			}
			agents, e := store.ListOnlineHostAgentIDs(ctx, source.ProjectID, source.HostGroupRef, spec.OS)
			if e != nil {
				return nil, e
			}
			if len(agents) == 0 {
				return nil, fmt.Errorf("目标主机组没有在线 Agent")
			}
			config["frozen_host"], err = json.Marshal(source)
			config["frozen_agent_ids"], _ = json.Marshal(agents)
		} else {
			store := &SQLHelmSourceStore{db: tx}
			source, e := store.LoadHelmSource(ctx, environmentID, sid, rid)
			if e != nil {
				return nil, e
			}
			if source == nil {
				return nil, fmt.Errorf("chart revision unavailable")
			}
			vars, e := store.LoadHelmVariables(ctx, source.ProjectID, environmentID, sid)
			if e != nil {
				return nil, e
			}
			for j := range vars {
				vars[j].CredentialData = ""
			}
			config["frozen_helm"], err = json.Marshal(source)
			config["frozen_helm_variables"], _ = json.Marshal(vars)
		}
		if err != nil {
			return nil, err
		}
		node.Config, err = json.Marshal(config)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

type sourceQuery interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func publicNodeSnapshot(raw json.RawMessage) json.RawMessage {
	var node map[string]json.RawMessage
	if json.Unmarshal(raw, &node) != nil {
		return nil
	}
	var config map[string]json.RawMessage
	if len(node["config"]) == 0 {
		return raw
	}
	if json.Unmarshal(node["config"], &config) != nil {
		return nil
	}
	delete(config, "frozen_yaml")
	delete(config, "frozen_helm")
	delete(config, "frozen_helm_variables")
	delete(config, "frozen_host")
	delete(config, "frozen_agent_ids")
	node["config"], _ = json.Marshal(config)
	result, _ := json.Marshal(node)
	return result
}
