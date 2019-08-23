package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/cmd/dandelion/registry"
	"github.com/tengattack/dandelion/log"
	appsv1 "k8s.io/api/apps/v1"
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
	Name      string `json:"name"`
	ImageName string `json:"image_name"`
	Image     string `json:"image"`
	Replicas  int    `json:"replicas"`
	Revision  int64  `json:"revision"`
}

// DeploymentEvent for deployment status
type DeploymentEvent struct {
	Name   string                              `json:"name"`
	Action string                              `json:"action"`
	Event  string                              `json:"event"`
	Status *extensionsv1beta1.DeploymentStatus `json:"status"`
}

// consts
const (
	RevisionAnnotation    = "deployment.kubernetes.io/revision"
	DandelionManagedLabel = "dandelion-managed"
)

var (
	clientset         *kubernetes.Clientset
	deploymentsClient typedappsv1.DeploymentInterface
	registryClient    *registry.Client
	eventsConns       map[string][]*websocket.Conn
	eventsConnMutex   *sync.Mutex

	errDeploymentIsNotManaged = errors.New("deployment is not managed by dandelion")
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
	eventsConnMutex = new(sync.Mutex)
	eventsConns = make(map[string][]*websocket.Conn)
	return nil
}

func getImageName(dp *appsv1.Deployment) string {
	if image, ok := dp.Labels["image"]; ok {
		return image
	}
	return dp.Name
}

func isManaged(dp *appsv1.Deployment) bool {
	_, ok := dp.Labels[DandelionManagedLabel]
	return ok
}

func getDeployment(dp *appsv1.Deployment) *Deployment {
	revision, _ := strconv.ParseInt(dp.Annotations[RevisionAnnotation], 10, 64)
	d := Deployment{
		Name:      dp.Name,
		ImageName: getImageName(dp),
		Replicas:  int(*dp.Spec.Replicas),
		Revision:  revision,
	}
	if len(dp.Spec.Template.Spec.Containers) > 0 {
		d.Image = dp.Spec.Template.Spec.Containers[0].Image
	}
	return &d
}

func kubeListHandler(c *gin.Context) {
	list, err := deploymentsClient.List(metav1.ListOptions{})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment list error: %v", err))
		return
	}

	ds := make([]*Deployment, 0, len(list.Items))
	for _, dp := range list.Items {
		// check permissions
		if !isManaged(&dp) {
			continue
		}

		d := getDeployment(&dp)
		ds = append(ds, d)
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

	var dp *appsv1.Deployment
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		result, getErr := deploymentsClient.Get(deployment, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		// check permissions
		if !isManaged(result) {
			return errDeploymentIsNotManaged
		}

		// get image name from labels
		imageName := getImageName(result)
		image := fmt.Sprintf("%s/%s:%s", u.Host, imageName, tag)

		result.Spec.Template.Spec.Containers[0].Image = image // change image
		var updateErr error
		dp, updateErr = deploymentsClient.Update(result)
		return updateErr
	})
	if retryErr == errDeploymentIsNotManaged {
		abortWithError(c, http.StatusForbidden, retryErr.Error())
		return
	}
	if retryErr != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment set-image error: %v", retryErr))
		return
	}

	// trigger events
	triggerDeploymentEvent(deployment, "setversiontag")

	succeed(c, gin.H{"deployment": getDeployment(dp), "ok": 1})
}

func kubeRollbackHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	dp, err := deploymentsClient.Get(deployment, metav1.GetOptions{})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment get error: %v", err))
		return
	}
	// check permissions
	if !isManaged(dp) {
		abortWithError(c, http.StatusForbidden, errDeploymentIsNotManaged.Error())
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

	// trigger events
	triggerDeploymentEvent(deployment, "rollback")

	var d *Deployment
	dpNew, err := deploymentsClient.Get(deployment, metav1.GetOptions{})
	if err != nil {
		log.LogError.Errorf("deployment get after rollback error: %v", err)
		// PASS
	} else {
		d = getDeployment(dpNew)
	}

	succeed(c, gin.H{"deployment": d, "ok": 1})
}

func kubeListTagsHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	dp, err := deploymentsClient.Get(deployment, metav1.GetOptions{})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment get error: %v", err))
		return
	}
	// check permissions
	if !isManaged(dp) {
		abortWithError(c, http.StatusForbidden, errDeploymentIsNotManaged.Error())
		return
	}

	// get image name from labels
	imageName := getImageName(dp)
	tags, err := registryClient.ListTags(imageName)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("registry list tags error: %v", err))
		return
	}

	sort.Sort(sort.Reverse(sort.StringSlice(tags.Tags)))

	succeed(c, gin.H{"image_name": tags.Name, "tags": tags.Tags})
}

func startNewConn(deployment string, conn *websocket.Conn) {
	eventsConnMutex.Lock()
	defer eventsConnMutex.Unlock()

	conns, ok := eventsConns[deployment]
	if !ok {
		conns = make([]*websocket.Conn, 0, 5)
	}
	eventsConns[deployment] = append(conns, conn)
}

func endConn(deployment string, conn *websocket.Conn) {
	eventsConnMutex.Lock()
	defer eventsConnMutex.Unlock()

	conns, ok := eventsConns[deployment]
	if ok {
		for i := 0; i < len(conns); i++ {
			if conns[i] == conn {
				// order is not important
				conns[i] = conns[len(conns)-1]
				eventsConns[deployment] = conns[:len(conns)-1]
				log.LogAccess.Debugf("events connections %s remove index %d", deployment, i)
				break
			}
		}
	}
}

func publishEventTo(deployment string, event *DeploymentEvent) {
	eventsConnMutex.Lock()
	defer eventsConnMutex.Unlock()

	conns, ok := eventsConns[deployment]
	if ok {
		for _, conn := range conns {
			go conn.WriteJSON(event)
		}
	}
}

// https://github.com/kubernetes/kubernetes/blob/74bcefc8b2bf88a2f5816336999b524cc48cf6c0/pkg/controller/deployment/util/deployment_util.go#L745
func isDeploymentComplete(deployment *extensionsv1beta1.Deployment, newStatus *extensionsv1beta1.DeploymentStatus) bool {
	return newStatus.UpdatedReplicas == *(deployment.Spec.Replicas) &&
		newStatus.Replicas == *(deployment.Spec.Replicas) &&
		newStatus.AvailableReplicas == *(deployment.Spec.Replicas) &&
		newStatus.ObservedGeneration >= deployment.Generation
}

func triggerDeploymentEvent(deployment, action string) {
	go func() {
		client := clientset.ExtensionsV1beta1().Deployments(config.Conf.Kubernetes.Namespace)
		timeoutCh := time.After(60 * time.Second)
		for {
			timeout := false
			select {
			case <-time.After(2 * time.Second):
			case <-timeoutCh:
				timeout = true
			}

			dp, err := client.Get(deployment, metav1.GetOptions{})
			if err != nil {
				log.LogError.Errorf("%s deployment get status error: %v", action, err)
				break
			}
			event := &DeploymentEvent{Name: deployment, Action: action, Event: "processing", Status: &dp.Status}
			if isDeploymentComplete(dp, &dp.Status) {
				event.Event = "complete"
				log.LogAccess.Infof("%s deployment %s completed", action, deployment)
			} else if timeout {
				event.Event = "timeout"
				log.LogAccess.Warnf("%s deployment %s timeout", action, deployment)
			}
			publishEventTo(deployment, event)

			if event.Event != "processing" {
				break
			}
		}
	}()
}

func kubeEventsHandler(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.LogError.Errorf("Failed to set websocket upgrade: %+v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer conn.Close()

	deployment := c.Param("deployment")

	startNewConn(deployment, conn)
	defer endConn(deployment, conn)
	defer conn.Close()

	for {
		t, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.LogError.Errorf("Unexpected close error: %v", err)
			}
			break
		}
		if t == websocket.TextMessage || t == websocket.BinaryMessage {
			// TODO: ping pong
		}
	}
}
