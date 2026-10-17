// ci-node-check reads CI node health and, when explicitly selected, kernel
// limits from an existing container. It never creates pods or changes sysctls.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/build"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/db"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
	"github.com/zeromicro/go-zero/core/conf"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

func main() {
	configFile := flag.String("config", "configs/platform-api.yaml", "platform config")
	nodeName := flag.String("node", "", "node to inspect")
	podName := flag.String("pod", "", "existing running pod for read-only kernel inspection")
	namespace := flag.String("namespace", "kube-system", "pod namespace")
	container := flag.String("container", "", "container in selected pod")
	flag.Parse()
	if err := run(*configFile, *nodeName, *podName, *namespace, *container); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(configFile, nodeName, podName, namespace, container string) error {
	var cfg config.PlatformAPIConfig
	if err := conf.Load(configFile, &cfg); err != nil {
		return err
	}
	database, err := db.NewMySQL(&cfg.MySQL)
	if err != nil {
		return fmt.Errorf("database connection failed")
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	settings, err := build.NewRepository(database).FindPlatformBuildSettings(ctx)
	if err != nil {
		return err
	}
	if settings == nil || settings.ClusterID <= 0 {
		return fmt.Errorf("CI cluster is not configured")
	}
	box, err := secretbox.New(config.CredentialEncryptionKey(cfg.CredentialEncryptionKey))
	if err != nil {
		return fmt.Errorf("credential key is unavailable")
	}
	factory := workflow.NewSQLKubernetesClientFactory(workflow.NewSQLClusterConnectionStore(database), box, workflow.KubernetesFactoryOptions{Timeout: 15 * time.Second})
	restConfig, err := factory.RESTConfig(ctx, settings.ClusterID)
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	fmt.Printf("CI cluster=%d namespace=%s builder=%s\n", settings.ClusterID, settings.Namespace, settings.BuilderImage)
	if nodeName == "" {
		return fmt.Errorf("specify -node from the failed build log")
	}
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("node=%s kernel=%s os=%s\n", node.Name, node.Status.NodeInfo.KernelVersion, node.Status.NodeInfo.OSImage)
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady || condition.Type == corev1.NodeDiskPressure || condition.Type == corev1.NodeMemoryPressure || condition.Type == corev1.NodePIDPressure {
			fmt.Printf("%s=%s reason=%s\n", condition.Type, condition.Status, condition.Reason)
		}
	}
	if podName == "" {
		pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "spec.nodeName=" + nodeName})
		if err != nil {
			return err
		}
		for _, pod := range pods.Items {
			if pod.Namespace != settings.Namespace && pod.Namespace != "kube-system" {
				continue
			}
			fmt.Printf("pod=%s/%s phase=%s", pod.Namespace, pod.Name, pod.Status.Phase)
			for _, c := range pod.Spec.Containers {
				fmt.Printf(" container=%s", c.Name)
			}
			fmt.Println()
		}
		return nil
	}
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if pod.Spec.NodeName != nodeName || pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("selected pod must be running on the requested node")
	}
	command := `for f in /proc/sys/user/max_user_namespaces /proc/sys/kernel/unprivileged_userns_clone /proc/sys/kernel/apparmor_restrict_unprivileged_userns; do if [ -r "$f" ]; then printf '%s=' "$f"; cat "$f"; fi; done; df -h / /tmp; df -i / /tmp`
	request := client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(namespace).SubResource("exec").VersionedParams(&corev1.PodExecOptions{Container: container, Command: []string{"sh", "-c", command}, Stdout: true, Stderr: true}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", request.URL())
	if err != nil {
		return err
	}
	return executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: os.Stdout, Stderr: os.Stderr})
}
