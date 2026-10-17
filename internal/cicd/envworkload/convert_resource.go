package envworkload

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func podToItem(p *corev1.Pod) PodItem {
	total := len(p.Spec.Containers)
	readyCount := 0
	var restarts int32
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Ready {
			readyCount++
		}
		restarts += cs.RestartCount
	}
	containers := make([]string, 0, total)
	images := make([]string, 0, total)
	for _, c := range p.Spec.Containers {
		containers = append(containers, c.Name)
		images = append(images, c.Image)
	}
	for _, c := range p.Spec.InitContainers {
		containers = append(containers, c.Name)
	}
	for _, c := range p.Spec.EphemeralContainers {
		containers = append(containers, c.Name)
	}
	status := string(p.Status.Phase)
	if p.Status.Phase == corev1.PodRunning {
		status = "NotReady"
		for _, condition := range p.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				status = "Ready"
			}
		}
	}
	for _, cs := range append(p.Status.InitContainerStatuses, p.Status.ContainerStatuses...) {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			status = cs.State.Waiting.Reason
			break
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			status = cs.State.Terminated.Reason
			if status == "" {
				status = "Error"
			}
			break
		}
	}
	if p.DeletionTimestamp != nil {
		status = "Terminating"
	}
	return PodItem{
		Name:       p.Name,
		Namespace:  p.Namespace,
		Phase:      string(p.Status.Phase),
		Status:     status,
		Ready:      fmt.Sprintf("%d/%d", readyCount, total),
		Restarts:   restarts,
		NodeName:   p.Spec.NodeName,
		PodIP:      p.Status.PodIP,
		Images:     images,
		Containers: containers,
		CreatedAt:  p.CreationTimestamp.UnixMilli(),
	}
}

func serviceToItem(s *corev1.Service) ServiceItem {
	ports := make([]string, 0, len(s.Spec.Ports))
	for _, port := range s.Spec.Ports {
		if port.NodePort > 0 {
			ports = append(ports, fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol))
		} else {
			ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
		}
	}
	var externalIP []string
	externalIP = append(externalIP, s.Spec.ExternalIPs...)
	for _, ing := range s.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			externalIP = append(externalIP, ing.IP)
		}
		if ing.Hostname != "" {
			externalIP = append(externalIP, ing.Hostname)
		}
	}
	return ServiceItem{
		Name:       s.Name,
		Namespace:  s.Namespace,
		Type:       string(s.Spec.Type),
		ClusterIP:  s.Spec.ClusterIP,
		ExternalIP: externalIP,
		Ports:      ports,
		CreatedAt:  s.CreationTimestamp.UnixMilli(),
	}
}

func ingressToItem(in *networkingv1.Ingress) IngressItem {
	hosts := make([]string, 0, len(in.Spec.Rules))
	for _, rule := range in.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	var address []string
	for _, ing := range in.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			address = append(address, ing.IP)
		}
		if ing.Hostname != "" {
			address = append(address, ing.Hostname)
		}
	}
	class := ""
	if in.Spec.IngressClassName != nil {
		class = *in.Spec.IngressClassName
	}
	return IngressItem{
		Name:      in.Name,
		Namespace: in.Namespace,
		Class:     class,
		Hosts:     hosts,
		Address:   address,
		CreatedAt: in.CreationTimestamp.UnixMilli(),
	}
}

func pvcToItem(p *corev1.PersistentVolumeClaim) PVCItem {
	capacity := ""
	if q, ok := p.Status.Capacity[corev1.ResourceStorage]; ok {
		capacity = q.String()
	} else if q, ok := p.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		capacity = q.String()
	}
	modes := make([]string, 0, len(p.Spec.AccessModes))
	for _, m := range p.Spec.AccessModes {
		modes = append(modes, string(m))
	}
	sc := ""
	if p.Spec.StorageClassName != nil {
		sc = *p.Spec.StorageClassName
	}
	return PVCItem{
		Name:         p.Name,
		Namespace:    p.Namespace,
		Status:       string(p.Status.Phase),
		Capacity:     capacity,
		AccessModes:  strings.Join(modes, ","),
		StorageClass: sc,
		VolumeName:   p.Spec.VolumeName,
		CreatedAt:    p.CreationTimestamp.UnixMilli(),
	}
}

func configMapToItem(c *corev1.ConfigMap) ConfigItem {
	return ConfigItem{
		Name:      c.Name,
		Namespace: c.Namespace,
		Kind:      "ConfigMap",
		Keys:      len(c.Data) + len(c.BinaryData),
		CreatedAt: c.CreationTimestamp.UnixMilli(),
	}
}

func secretToItem(s *corev1.Secret) ConfigItem {
	return ConfigItem{
		Name:      s.Name,
		Namespace: s.Namespace,
		Kind:      "Secret",
		Type:      string(s.Type),
		Keys:      len(s.Data),
		CreatedAt: s.CreationTimestamp.UnixMilli(),
	}
}
