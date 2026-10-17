package assetdeploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// Check dependencies before writing any workloads. Resources in this manifest
// and Secrets prepared by this deployment are available in the same operation.
// Pending PVCs are allowed: WaitForFirstConsumer needs a scheduled Pod to bind.
func preflightDependencies(ctx context.Context, client kubernetes.Interface, target workflow.YAMLTarget) error {
	if len(target.Manifest) == 0 {
		return nil
	}
	objects, err := workflow.ValidateYAMLManifest(target.Manifest, target.Namespace)
	if err != nil {
		return err
	}
	configMaps := map[string]*corev1.ConfigMap{}
	claims := map[string]*corev1.PersistentVolumeClaim{}
	secrets := map[string]*corev1.Secret{}
	for _, ref := range target.AssetSecrets {
		if secrets[ref.Name] == nil {
			secrets[ref.Name] = &corev1.Secret{Data: map[string][]byte{}}
		}
		secrets[ref.Name].Data[ref.Key] = nil
	}
	for _, obj := range objects {
		switch obj.GetKind() {
		case "ConfigMap":
			var item corev1.ConfigMap
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &item); err != nil {
				return fmt.Errorf("ConfigMap %s 格式错误", obj.GetName())
			}
			configMaps[item.Name] = &item
		case "PersistentVolumeClaim":
			var item corev1.PersistentVolumeClaim
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &item); err != nil {
				return fmt.Errorf("PVC %s 格式错误", obj.GetName())
			}
			claims[item.Name] = &item
			if item.Spec.StorageClassName != nil && *item.Spec.StorageClassName != "" {
				if _, err := client.StorageV1().StorageClasses().Get(ctx, *item.Spec.StorageClassName, metav1.GetOptions{}); err != nil {
					return dependencyError("StorageClass", *item.Spec.StorageClassName, target.Namespace, err)
				}
			}
		}
	}
	checkSecret := func(name, key string, optional bool) error {
		item := secrets[name]
		if item == nil {
			item, err = client.CoreV1().Secrets(target.Namespace).Get(ctx, name, metav1.GetOptions{})
			if optional && apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return dependencyError("Secret", name, target.Namespace, err)
			}
			secrets[name] = item
		}
		if _, exists := item.Data[key]; key != "" && !exists && !optional {
			return fmt.Errorf("命名空间 %s 的 Secret %s 缺少键 %s", target.Namespace, name, key)
		}
		return nil
	}
	checkConfig := func(name, key string, optional bool) error {
		item := configMaps[name]
		if item == nil {
			item, err = client.CoreV1().ConfigMaps(target.Namespace).Get(ctx, name, metav1.GetOptions{})
			if optional && apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return dependencyError("ConfigMap", name, target.Namespace, err)
			}
			configMaps[name] = item
		}
		_, text := item.Data[key]
		_, binary := item.BinaryData[key]
		if key != "" && !text && !binary && !optional {
			return fmt.Errorf("命名空间 %s 的 ConfigMap %s 缺少键 %s", target.Namespace, name, key)
		}
		return nil
	}
	for _, obj := range objects {
		if obj.GetKind() == "Ingress" {
			var ingress networkingv1.Ingress
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &ingress); err != nil {
				return fmt.Errorf("Ingress %s 格式错误", obj.GetName())
			}
			if ingress.Spec.IngressClassName != nil {
				name := *ingress.Spec.IngressClassName
				if _, err := client.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{}); err != nil {
					return dependencyError("IngressClass", name, target.Namespace, err)
				}
			}
			for _, tls := range ingress.Spec.TLS {
				if tls.SecretName == "" {
					continue
				}
				for _, key := range []string{corev1.TLSCertKey, corev1.TLSPrivateKeyKey} {
					if err := checkSecret(tls.SecretName, key, false); err != nil {
						return err
					}
				}
			}
		}
		podPath := []string{"spec", "template", "spec"}
		if obj.GetKind() == "CronJob" {
			podPath = []string{"spec", "jobTemplate", "spec", "template", "spec"}
		}
		podData, found, err := unstructured.NestedMap(obj.Object, podPath...)
		if err != nil {
			return fmt.Errorf("%s %s Pod 格式错误", obj.GetKind(), obj.GetName())
		}
		if !found {
			continue
		}
		var pod corev1.PodSpec
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(podData, &pod); err != nil {
			return fmt.Errorf("%s %s Pod 格式错误", obj.GetKind(), obj.GetName())
		}
		for _, container := range append(pod.Containers, pod.InitContainers...) {
			if strings.Contains(container.Image, "axiops-config-validation.invalid/") {
				return fmt.Errorf("容器 %s 仍使用预览占位镜像，请先构建或选择真实产物镜像", container.Name)
			}
			for _, env := range container.Env {
				if strings.HasPrefix(env.Name, "SPRING_CLOUD_NACOS_") &&
					(strings.Contains(env.Value, "nacos.example.invalid") || env.Value == "replace-with-namespace-id") {
					return fmt.Errorf("容器 %s 的 %s 仍为 Nacos 示例值，请填写目标环境的实际配置", container.Name, env.Name)
				}
				if env.ValueFrom == nil {
					continue
				}
				if ref := env.ValueFrom.SecretKeyRef; ref != nil {
					if err := checkSecret(ref.Name, ref.Key, isOptional(ref.Optional)); err != nil {
						return err
					}
				}
				if ref := env.ValueFrom.ConfigMapKeyRef; ref != nil {
					if err := checkConfig(ref.Name, ref.Key, isOptional(ref.Optional)); err != nil {
						return err
					}
				}
			}
			for _, env := range container.EnvFrom {
				if ref := env.SecretRef; ref != nil {
					if err := checkSecret(ref.Name, "", isOptional(ref.Optional)); err != nil {
						return err
					}
				}
				if ref := env.ConfigMapRef; ref != nil {
					if err := checkConfig(ref.Name, "", isOptional(ref.Optional)); err != nil {
						return err
					}
				}
			}
		}
		for _, volume := range pod.Volumes {
			if ref := volume.PersistentVolumeClaim; ref != nil && claims[ref.ClaimName] == nil {
				claim, err := client.CoreV1().PersistentVolumeClaims(target.Namespace).Get(ctx, ref.ClaimName, metav1.GetOptions{})
				if err != nil {
					return dependencyError("PVC", ref.ClaimName, target.Namespace, err)
				}
				if claim.Status.Phase == corev1.ClaimLost || claim.DeletionTimestamp != nil {
					return fmt.Errorf("PVC %s 已丢失或正在删除", ref.ClaimName)
				}
				claims[ref.ClaimName] = claim
			}
			if ref := volume.ConfigMap; ref != nil {
				if err := checkVolumeKeys(volume.Name, ref.Name, ref.Items, isOptional(ref.Optional), pod, checkConfig); err != nil {
					return err
				}
			}
			if ref := volume.Secret; ref != nil {
				if err := checkVolumeKeys(volume.Name, ref.SecretName, ref.Items, isOptional(ref.Optional), pod, checkSecret); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkVolumeKeys(volume, name string, items []corev1.KeyToPath, optional bool, pod corev1.PodSpec, check func(string, string, bool) error) error {
	if err := check(name, "", optional); err != nil {
		return err
	}
	for _, item := range items {
		if err := check(name, item.Key, optional); err != nil {
			return err
		}
	}
	// With no items mapping the generated subPath is exactly the data key.
	if len(items) == 0 {
		for _, c := range append(pod.Containers, pod.InitContainers...) {
			for _, mount := range c.VolumeMounts {
				if mount.Name == volume && mount.SubPath != "" && mount.SubPathExpr == "" {
					if err := check(name, mount.SubPath, false); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
func isOptional(value *bool) bool { return value != nil && *value }
func dependencyError(kind, name, namespace string, err error) error {
	// Deliberately omit API bodies: they may contain credential data.
	return fmt.Errorf("部署依赖检查失败：%s %s（目标命名空间 %s）：%s", kind, name, namespace, apierrors.ReasonForError(err))
}
