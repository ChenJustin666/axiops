package serviceassets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func object(kind, name, ns string, sid int64, body map[string]interface{}) unstructured.Unstructured {
	version := "v1"
	if kind == "Ingress" {
		version = "networking.k8s.io/v1"
	}
	body["apiVersion"] = version
	body["kind"] = kind
	body["metadata"] = map[string]interface{}{"name": name, "namespace": ns, "labels": map[string]interface{}{"axiops.io/service-id": fmt.Sprint(sid), "axiops.io/managed-by": "service-config"}}
	return unstructured.Unstructured{Object: body}
}

// Apply produces ordinary Kubernetes resources and credential references only.
// Literal file contents are not interpreted as deployment template variables.
func Apply(input []unstructured.Unstructured, s Settings, sid int64, ns string) ([]unstructured.Unstructured, []SecretRef, error) {
	if err := Validate(s); err != nil {
		return nil, nil, err
	}
	if !s.Active() {
		return input, nil, nil
	}
	items := make([]unstructured.Unstructured, len(input))
	seen := map[string]bool{}
	for i := range input {
		items[i] = *input[i].DeepCopy()
		seen[items[i].GetKind()+"/"+items[i].GetName()] = true
	}
	add := func(item unstructured.Unstructured) error {
		key := item.GetKind() + "/" + item.GetName()
		if seen[key] {
			return fmt.Errorf("生成资源 %s 与已有模板冲突", key)
		}
		seen[key] = true
		items = append(items, item)
		return nil
	}
	var refs []SecretRef
	if s.Resources != nil || s.Replicas != nil || len(s.Command)+len(s.Args)+len(s.Files)+len(s.Secrets)+len(s.Volumes) > 0 {
		var targets []int
		for i, item := range items {
			kind := item.GetKind()
			if kind != "Deployment" && kind != "StatefulSet" && kind != "DaemonSet" && kind != "Job" && kind != "CronJob" {
				continue
			}
			if (s.WorkloadKind == "" || s.WorkloadKind == kind) && (s.WorkloadName == "" || s.WorkloadName == item.GetName()) {
				targets = append(targets, i)
			}
		}
		if len(targets) != 1 {
			return nil, nil, fmt.Errorf("请选择唯一的目标工作负载，多工作负载不能自动猜测")
		}
		index := targets[0]
		templatePath := []string{"spec", "template"}
		if items[index].GetKind() == "CronJob" {
			templatePath = []string{"spec", "jobTemplate", "spec", "template"}
		}
		template, ok, err := unstructured.NestedMap(items[index].Object, templatePath...)
		if err != nil || !ok {
			return nil, nil, fmt.Errorf("目标缺少Pod模板")
		}
		var pod corev1.PodTemplateSpec
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(template, &pod); err != nil {
			return nil, nil, fmt.Errorf("Pod模板格式无效")
		}
		ci := -1
		for i, c := range pod.Spec.Containers {
			if (s.Container == "" && len(pod.Spec.Containers) == 1) || s.Container == c.Name {
				ci = i
			}
		}
		if ci < 0 {
			return nil, nil, fmt.Errorf("请选择要配置的容器，多容器不能自动猜测")
		}
		container := &pod.Spec.Containers[ci]
		if s.Replicas != nil {
			kind := items[index].GetKind()
			if kind == "Job" || kind == "CronJob" {
				return nil, nil, fmt.Errorf("工作负载 %s 不支持副本数覆盖", kind)
			}
			if err := unstructured.SetNestedField(items[index].Object, int64(*s.Replicas), "spec", "replicas"); err != nil {
				return nil, nil, err
			}
		}
		if len(s.Command) > 0 {
			container.Command = append([]string(nil), s.Command...)
		}
		if len(s.Args) > 0 {
			container.Args = append([]string(nil), s.Args...)
		}
		if s.Resources != nil {
			if container.Resources.Requests == nil {
				container.Resources.Requests = corev1.ResourceList{}
			}
			if container.Resources.Limits == nil {
				container.Resources.Limits = corev1.ResourceList{}
			}
			for _, item := range []struct {
				name           corev1.ResourceName
				request, limit string
			}{
				{corev1.ResourceCPU, s.Resources.CPURequest, s.Resources.CPULimit},
				{corev1.ResourceMemory, s.Resources.MemoryRequest, s.Resources.MemoryLimit},
			} {
				if item.request != "" {
					container.Resources.Requests[item.name] = resource.MustParse(item.request)
				}
				if item.limit != "" {
					container.Resources.Limits[item.name] = resource.MustParse(item.limit)
				}
				req, hasReq := container.Resources.Requests[item.name]
				limit, hasLimit := container.Resources.Limits[item.name]
				if hasReq && hasLimit && req.Cmp(limit) > 0 {
					return nil, nil, fmt.Errorf("容器 %s 的 %s request 大于 limit（已合并模板与当前环境配置）", container.Name, item.name)
				}
			}
		}
		mount := func(name, key, dest string, source corev1.VolumeSource, ro bool) error {
			for _, v := range pod.Spec.Volumes {
				if v.Name == name {
					return fmt.Errorf("卷名 %s 与模板冲突", name)
				}
			}
			for _, m := range container.VolumeMounts {
				if m.MountPath == dest || pathOverlap(m.MountPath, dest) {
					return fmt.Errorf("挂载路径与模板已有挂载冲突")
				}
			}
			pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: name, VolumeSource: source})
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: name, MountPath: dest, SubPath: key, ReadOnly: ro})
			return nil
		}
		for _, f := range s.Files {
			name := f.ExistingConfigMap
			if name == "" {
				name = ResourceName(sid, "file", f.Name)
				if err = add(object("ConfigMap", name, ns, sid, map[string]interface{}{"data": map[string]interface{}{f.Filename: f.Content}})); err != nil {
					return nil, nil, err
				}
			}
			if err = mount("axiops-file-"+f.Name, f.Filename, f.MountPath, corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: name}}}, true); err != nil {
				return nil, nil, err
			}
		}
		groupNames := map[string]string{}
		for _, secret := range s.Secrets {
			if secret.Group == "" || secret.ExistingSecret != "" {
				continue
			}
			var fields []string
			for _, field := range s.Secrets {
				if field.Group == secret.Group {
					fields = append(fields, field.Key+":"+field.CredentialKey)
				}
			}
			sort.Strings(fields)
			version := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", secret.CredentialID, strings.Join(fields, "|"))))
			groupNames[secret.Group] = ResourceName(sid, "secret", secret.Group) + "-" + hex.EncodeToString(version[:6])
		}
		for _, secret := range s.Secrets {
			name := secret.ExistingSecret
			if name == "" {
				// A replacement credential must not overwrite a Secret used by old Pods.
				version := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s", secret.CredentialID, secret.CredentialKey, secret.Key)))
				name = ResourceName(sid, "secret", secret.Name) + "-" + hex.EncodeToString(version[:6])
				if secret.Group != "" {
					name = groupNames[secret.Group]
				}
				refs = append(refs, SecretRef{Name: name, Key: secret.Key, CredentialID: secret.CredentialID, CredentialKey: secret.CredentialKey})
			}
			if secret.EnvName != "" {
				for _, e := range container.Env {
					if e.Name == secret.EnvName {
						return nil, nil, fmt.Errorf("环境变量 %s 与模板冲突", secret.EnvName)
					}
				}
				container.Env = append(container.Env, corev1.EnvVar{Name: secret.EnvName, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: secret.Key}}})
			} else {
				if err = mount("axiops-secret-"+secret.Name, secret.Key, secret.MountPath, corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: name}}, true); err != nil {
					return nil, nil, err
				}
			}
		}
		for _, v := range s.Volumes {
			name := v.ExistingClaim
			if name == "" {
				name = ResourceName(sid, "pvc", v.Name)
				claimSpec := map[string]interface{}{"accessModes": []interface{}{v.AccessMode}, "resources": map[string]interface{}{"requests": map[string]interface{}{"storage": v.Capacity}}}
				// Omit the field to use the cluster's default StorageClass. An
				// explicit empty string disables dynamic provisioning in Kubernetes.
				if v.StorageClass != "" {
					claimSpec["storageClassName"] = v.StorageClass
				}
				if err = add(object("PersistentVolumeClaim", name, ns, sid, map[string]interface{}{"spec": claimSpec})); err != nil {
					return nil, nil, err
				}
			}
			if err = mount("axiops-pvc-"+v.Name, "", v.MountPath, corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: name, ReadOnly: v.ReadOnly}}, v.ReadOnly); err != nil {
				return nil, nil, err
			}
		}
		// A new configuration revision rolls Pods on the next explicit deployment.
		encoded, _ := json.Marshal(s)
		sum := sha256.Sum256(encoded)
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		pod.Annotations["axiops.io/service-config-hash"] = hex.EncodeToString(sum[:])
		converted, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pod)
		if err != nil {
			return nil, nil, err
		}
		if err = unstructured.SetNestedMap(items[index].Object, converted, templatePath...); err != nil {
			return nil, nil, err
		}
	}
	for _, r := range s.Routes {
		matched := false
		for _, item := range items {
			if item.GetKind() != "Service" || item.GetName() != r.Service {
				continue
			}
			var svc corev1.Service
			if runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &svc) == nil {
				for _, p := range svc.Spec.Ports {
					if p.Port == int32(r.Port) {
						matched = true
					}
				}
			}
		}
		if !matched {
			return nil, nil, fmt.Errorf("路由目标 %s:%d 不在当前服务模板中", r.Service, r.Port)
		}
		spec := map[string]interface{}{"rules": []interface{}{map[string]interface{}{"host": r.Host, "http": map[string]interface{}{"paths": []interface{}{map[string]interface{}{"path": r.Path, "pathType": routePathType(r), "backend": map[string]interface{}{"service": map[string]interface{}{"name": r.Service, "port": map[string]interface{}{"number": int64(r.Port)}}}}}}}}}
		if r.Class != "" {
			spec["ingressClassName"] = r.Class
		}
		if r.TLSEnabled || r.TLSSecret != "" {
			hosts := []interface{}{r.Host}
			if len(r.TLSHosts) > 0 {
				hosts = nil
				for _, host := range r.TLSHosts {
					hosts = append(hosts, host)
				}
			}
			tls := map[string]interface{}{"hosts": hosts}
			if r.TLSSecret != "" {
				tls["secretName"] = r.TLSSecret
			}
			spec["tls"] = []interface{}{tls}
		}
		ingress := object("Ingress", ResourceName(sid, "ingress", r.Name), ns, sid, map[string]interface{}{"spec": spec})
		ingress.SetAnnotations(r.Annotations)
		labels := ingress.GetLabels()
		for key, value := range r.Labels {
			labels[key] = value
		}
		ingress.SetLabels(labels)
		if err := add(ingress); err != nil {
			return nil, nil, err
		}
	}
	return items, refs, nil
}
func pathOverlap(a, b string) bool {
	return len(a) > 0 && len(b) > 0 && (strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/"))
}

// Exported for the credential preparer without exposing secret content.
func SecretMetadata(name, ns string, sid int64) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: ns, Labels: map[string]string{"axiops.io/service-id": fmt.Sprint(sid), "axiops.io/managed-by": "service-config"}}
}
