package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	k *client.Clientset
}

func New(cluster string) *Client {
	k8sConfig, err := BuildConfigFromFlags(cluster, os.Getenv("KUBECONFIG"))
	if err != nil {
		log.WithError(err).Fatal("initialize kubeconfig")
	}

	clientSet, err := client.NewForConfig(k8sConfig)
	if err != nil {
		log.WithError(err).Fatal("initialize kubernetes client")
	}
	return &Client{k: clientSet}
}

func (c *Client) NewNode(ctx context.Context, nodeName string) bool {
	return c.getNode(ctx, nodeName) == nil
}

func (c *Client) WaitForNode(ctx context.Context, nodeName string) {
	log.WithField("node", nodeName).Infof("wait for node to join cluster")
	retry(ctx, 10, func() error {
		node := c.getNode(ctx, nodeName)
		if node == nil {
			return fmt.Errorf("node %s not found", nodeName)
		}

		return nil
	})
}

func (c *Client) LabelNode(ctx context.Context, nodeName, key, value string) bool {
	log.WithField("node", nodeName).Infof("label node")
	patch := []byte(fmt.Sprintf(`{"metadata":{"labels":{%q:%q}}}`, key, value))
	retry(ctx, 2, func() error {
		_, err := c.k.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	})
	return true
}

func (c *Client) Drain(ctx context.Context, nodeName string) {
	log.WithField("node", nodeName).Infof("initiate node drain")
	retry(ctx, 2, func() error {
		node := c.getNode(ctx, nodeName)
		node.Spec.Unschedulable = true

		if !hasTaint("nais.io/nitro-shutdown", node.Spec.Taints) {
			node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
				Key:    "nais.io/nitro-shutdown",
				Value:  "true",
				Effect: corev1.TaintEffectNoExecute,
			})
		}

		if !hasTaint("nais.io/flannel-unavailable", node.Spec.Taints) {
			node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
				Key:    "nais.io/flannel-unavailable",
				Value:  "true",
				Effect: corev1.TaintEffectNoSchedule,
			})
		}
		if _, err := c.k.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
			return err
		}
		return nil
	})
}

func (c *Client) nodePods(ctx context.Context, nodeName string) []corev1.Pod {
	var pods []corev1.Pod
	retry(ctx, 2, func() error {
		resp, err := c.k.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
		})
		if err != nil {
			return err
		}

		pods = resp.Items
		return nil
	})

	return pods
}

func (c *Client) getNodes(ctx context.Context) []corev1.Node {
	var nodes []corev1.Node
	timeoutSeconds := int64(2)
	retry(ctx, 5, func() error {
		resp, err := c.k.CoreV1().Nodes().List(ctx, metav1.ListOptions{TimeoutSeconds: &timeoutSeconds})
		if err != nil {
			return err
		}
		nodes = resp.Items
		return nil
	})
	return nodes
}

func (c *Client) getNode(ctx context.Context, nodeName string) *corev1.Node {
	for _, n := range c.getNodes(ctx) {
		if n.Name == nodeName {
			return &n
		}
	}
	return nil
}

func hasTaint(key string, taints []corev1.Taint) bool {
	for _, t := range taints {
		if t.Key == key {
			return true
		}
	}
	return false
}

func (c *Client) isDaemonSet(ctx context.Context, podName string) bool {
	var found bool
	retry(ctx, 5, func() error {
		resp, err := c.k.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, ds := range resp.Items {
			if strings.Contains(podName, ds.Name) {
				found = true
				return nil
			}
		}
		found = false
		return nil
	})

	return found
}

func (c *Client) DeleteNode(ctx context.Context, nodeName string) {
	retry(ctx, 2, func() error {
		return c.k.CoreV1().Nodes().Delete(ctx, nodeName, metav1.DeleteOptions{})
	})
}

func (c *Client) Wait(ctx context.Context, nodeName string) {
	maxWaitMinutes := 3
	endTime := time.Now().Add(time.Duration(maxWaitMinutes) * 55 * time.Second)
	log.WithField("node", nodeName).Infof("will wait until %s", endTime.Local().String())
	retry(ctx, maxWaitMinutes, func() error {
		remaining := len(c.remainingPods(ctx, nodeName))
		if time.Now().After(endTime) {
			log.WithField("node", nodeName).Warnf("abort wait on drain because of timeout")
			return nil
		}

		if remaining > 0 {
			return fmt.Errorf("pods remaining: %d", remaining)
		}

		log.WithField("node", nodeName).Infof("no pods left")
		return nil
	})
}

func (c *Client) remainingPods(ctx context.Context, node string) []string {
	pods := c.nodePods(ctx, node)

	var remaining []string

	for _, pod := range pods {
		if c.isDaemonSet(ctx, pod.Name) {
			continue
		}

		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}

		remaining = append(remaining, pod.Name)
	}
	return remaining
}

func retry(ctx context.Context, maxWaitMinutes int, f func() error) {
	minutes := time.Duration(maxWaitMinutes)
	maxWaitTime := minutes * time.Minute
	const RetryInterval = 5 * time.Second

	retryTicker := time.NewTicker(RetryInterval)
	ctx, cancel := context.WithTimeout(ctx, maxWaitTime)
	defer cancel()

	var log log.FieldLogger = log.StandardLogger()
	panicExtra := ""
	if name := GetName(ctx); name != "" {
		log = log.WithField("node", name)
		panicExtra = fmt.Sprintf(" (node: %s)", name)
	}

	for {
		err := f()
		if err == nil {
			return
		}

		log.Infof("retrying in %s (max wait time: %s): %s", RetryInterval.String(), maxWaitTime.String(), err)

		select {
		case <-ctx.Done():
			panic(fmt.Sprintf("stopping retry: retry context is done: %s%s", ctx.Err(), panicExtra))
		case <-retryTicker.C:
		}
	}
}

func BuildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

type ctxKey string

const ctxName ctxKey = "name"

func WithName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, ctxName, name)
}

func GetName(ctx context.Context) string {
	r, ok := ctx.Value(ctxName).(string)
	if !ok {
		return ""
	}
	return r
}
