package envworkload

import (
	"context"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

// ListServices returns service rows.
func (l *Logic) ListServices(ctx context.Context, userID, environmentID int64) ([]ServiceItem, int) {
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	svcs, err := c.listServices(ctx, ns)
	if err != nil {
		return nil, resourceReadCode(err)
	}
	items := make([]ServiceItem, 0, len(svcs))
	for i := range svcs {
		items = append(items, serviceToItem(&svcs[i]))
	}
	return items, errcode.CodeSuccess
}

// ListIngresses returns ingress rows.
func (l *Logic) ListIngresses(ctx context.Context, userID, environmentID int64) ([]IngressItem, int) {
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	ings, err := c.listIngresses(ctx, ns)
	if err != nil {
		return nil, resourceReadCode(err)
	}
	items := make([]IngressItem, 0, len(ings))
	for i := range ings {
		items = append(items, ingressToItem(&ings[i]))
	}
	return items, errcode.CodeSuccess
}

// ListPVCs returns persistent volume claim rows.
func (l *Logic) ListPVCs(ctx context.Context, userID, environmentID int64) ([]PVCItem, int) {
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	pvcs, err := c.listPVCs(ctx, ns)
	if err != nil {
		return nil, resourceReadCode(err)
	}
	items := make([]PVCItem, 0, len(pvcs))
	for i := range pvcs {
		items = append(items, pvcToItem(&pvcs[i]))
	}
	return items, errcode.CodeSuccess
}

// ListConfigs returns ConfigMap or Secret rows depending on kind.
func (l *Logic) ListConfigs(ctx context.Context, userID, environmentID int64, kind string) ([]ConfigItem, int) {
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	var items []ConfigItem
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "configmap":
		cms, err := c.listConfigMaps(ctx, ns)
		if err != nil {
			return nil, resourceReadCode(err)
		}
		for i := range cms {
			items = append(items, configMapToItem(&cms[i]))
		}
	case "secret":
		secrets, err := c.listSecrets(ctx, ns)
		if err != nil {
			return nil, resourceReadCode(err)
		}
		for i := range secrets {
			items = append(items, secretToItem(&secrets[i]))
		}
	default:
		return nil, errcode.CodeParamInvalid
	}
	if items == nil {
		items = []ConfigItem{}
	}
	return items, errcode.CodeSuccess
}
