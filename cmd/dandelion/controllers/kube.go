package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/cmd/dandelion/registry"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

// Deployment for kube deployment
type Deployment struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Replicas int    `json:"replicas"`
	Revision int64  `json:"revision"`
}

// consts
const (
	RevisionAnnotation = "deployment.kubernetes.io/revision"
)

var (
	clientset         *kubernetes.Clientset
	deploymentsClient typedappsv1.DeploymentInterface
	registryClient    *registry.Client
)

func initKubeClient() error {
	var kubeConfig *rest.Config
	var err error
	if config.Conf.Kubernetes.InCluster {
		kubeConfig, err = rest.InClusterConfig()
	} else {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.Conf.Kubernetes.Config)
	}
	if err != nil {
		return err
	}
	clientset, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	deploymentsClient = clientset.AppsV1().Deployments(config.Conf.Kubernetes.Namespace)
	registryClient = registry.NewClient(&config.Conf.Registry)
	return nil
}

func kubeListHandler(c *gin.Context) {
	list, err := deploymentsClient.List(metav1.ListOptions{})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment list error: %v", err))
		return
	}

	// TODO: check permissions
	ds := make([]Deployment, len(list.Items))
	for i, d := range list.Items {
		revision, _ := strconv.ParseInt(d.Annotations[RevisionAnnotation], 10, 64)
		ds[i] = Deployment{Name: d.Name, Replicas: int(*d.Spec.Replicas), Revision: revision}
		if len(d.Spec.Template.Spec.Containers) > 0 {
			ds[i].Image = d.Spec.Template.Spec.Containers[0].Image
		}
	}

	succeed(c, gin.H{"deployments": ds})
}

func kubeSetVersionTagHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	tag := c.PostForm("version_tag")
	// TODO: check tag exists

	u, err := url.Parse(config.Conf.Registry.Endpoint)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("registry endpoint error: %v", err))
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		dp, getErr := deploymentsClient.Get(deployment, metav1.GetOptions{})
		// TODO: check permissions
		if getErr != nil {
			return getErr
		}

		// TODO: get registry from labels
		image := fmt.Sprintf("%s/%s:%s", u.Host, deployment, tag)

		dp.Spec.Template.Spec.Containers[0].Image = image // change image
		_, updateErr := deploymentsClient.Update(dp)
		return updateErr
	})
	if retryErr != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment set-image error: %v", retryErr))
		return
	}

	// TODO: trigger events

	succeed(c, gin.H{"ok": 1})
}

func kubeRollbackHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	dp, err := deploymentsClient.Get(deployment, metav1.GetOptions{})
	// TODO: check permissions
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment get error: %v", err))
		return
	}

	revision, _ := strconv.ParseInt(dp.Annotations[RevisionAnnotation], 10, 64)
	if revision <= 1 {
		abortWithError(c, http.StatusBadRequest, "deployment no enough revision")
		return
	}

	dr := new(extensionsv1beta1.DeploymentRollback)
	dr.Name = dp.Name
	// dr.UpdatedAnnotations = annotations
	dr.RollbackTo = extensionsv1beta1.RollbackConfig{Revision: revision - 1}

	// Rollback
	err = clientset.ExtensionsV1beta1().Deployments(config.Conf.Kubernetes.Namespace).Rollback(dr)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment rollback error: %v", err))
		return
	}

	// TODO: trigger events

	succeed(c, gin.H{"ok": 1})
}

func kubeListTagsHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	_, err := deploymentsClient.Get(deployment, metav1.GetOptions{})
	// TODO: check permissions
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment get error: %v", err))
		return
	}

	// TODO: get registry from labels
	tags, err := registryClient.ListTags(deployment)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment get error: %v", err))
		return
	}

	sort.Sort(sort.Reverse(sort.StringSlice(tags.Tags)))

	succeed(c, tags)
}
