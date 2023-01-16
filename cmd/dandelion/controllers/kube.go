package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tengattack/dandelion/cmd/dandelion/cloudprovider"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/dandelion/cmd/dandelion/registry"
	"github.com/tengattack/dandelion/cmd/dandelion/webhook"
	"github.com/tengattack/dandelion/log"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedautoscalingv2beta2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

// PatchStringValue specifies a patch operation for a string.
type PatchStringValue struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// Deployment for kube deployment
type Deployment struct {
	Name      string `json:"name"`
	ImageName string `json:"image_name"`
	Image     string `json:"image"`
	Replicas  int    `json:"replicas"`
	Revision  int64  `json:"revision"`
}

// HPA for kube hpa
type HPA struct {
	Name        string `json:"name"`
	MinReplicas int    `json:"min_replicas"`
	MaxReplicas int    `json:"max_replicas"`
}

// DeploymentEvent for deployment status
type DeploymentEvent struct {
	Name   string                              `json:"name"`
	Action string                              `json:"action"`
	Event  string                              `json:"event"`
	Status *extensionsv1beta1.DeploymentStatus `json:"status"`
}

// NodeNameCache for new node name
type NodeNameCache struct {
	lock     sync.Mutex
	names    map[string]struct{}
	lastTime time.Time
}

// Equal checks whether the event is same
func (e *DeploymentEvent) Equal(event *DeploymentEvent) bool {
	ok := false
	if event != nil {
		ok = e.Name == event.Name &&
			e.Action == event.Action &&
			e.Event == event.Event
		if ok {
			if e.Status == nil && event.Status == nil {
				// no status
				// PASS
			} else if e.Status != nil && event.Status != nil {
				ok = e.Status.String() == event.Status.String()
			} else {
				// one of event has no status
				ok = false
			}
		}
	}
	return ok
}

// consts
const (
	RevisionAnnotation    = "deployment.kubernetes.io/revision"
	DandelionManagedLabel = "dandelion.to/managed"
	LastRestartEnv        = "LAST_RESTART"
)

var (
	clientset         *kubernetes.Clientset
	deploymentsClient typedappsv1.DeploymentInterface
	hpasClient        typedautoscalingv2beta2.HorizontalPodAutoscalerInterface
	registryClient    *registry.Client
	webhookClient     *webhook.Client
	eventsConns       map[string][]*websocket.Conn
	eventsConnMutex   *sync.Mutex
	nodeNameCache     *NodeNameCache

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
	hpasClient = clientset.AutoscalingV2beta2().HorizontalPodAutoscalers(config.Conf.Kubernetes.Namespace)
	registryClient = registry.NewClient(&config.Conf.Registry)
	webhookClient = webhook.NewClient(&config.Conf.Webhook)
	eventsConnMutex = new(sync.Mutex)
	eventsConns = make(map[string][]*websocket.Conn)
	nodeNameCache = new(NodeNameCache)
	return nil
}

func getRegistryBind(dp *appsv1.Deployment) string {
	if image, ok := dp.Labels["dandelion.to/bind"]; ok {
		image = strings.ReplaceAll(image, "__", "/")
		return image
	}
	return dp.Name
}

func getImageName(dp *appsv1.Deployment) string {
	if image, ok := dp.Labels["dandelion.to/image"]; ok {
		image = strings.ReplaceAll(image, "__", "/")
		return image
	}
	if len(dp.Spec.Template.Spec.Containers) > 0 {
		image := dp.Spec.Template.Spec.Containers[0].Image
		pos := strings.LastIndex(image, ":")
		if pos >= 0 {
			return image[:pos]
		}
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

func getHPA(hpa *v2beta2.HorizontalPodAutoscaler) *HPA {
	if hpa == nil {
		return nil
	}
	h := HPA{
		Name:        hpa.Name,
		MaxReplicas: int(hpa.Spec.MaxReplicas),
	}
	if hpa.Spec.MinReplicas != nil {
		h.MinReplicas = int(*hpa.Spec.MinReplicas)
	}
	return &h
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

func kubeDetailHandler(c *gin.Context) {
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

	hpa, err := hpasClient.Get(deployment, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		hpa = nil
		err = nil
	}
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("hpa get error: %v", err))
		return
	}

	d := getDeployment(dp)
	h := getHPA(hpa)

	succeed(c, gin.H{"deployment": d, "hpa": h})
}

func kubeSetVersionTagHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	tag := c.PostForm("version_tag")
	// TODO: check tag exists

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
		image := fmt.Sprintf("%s:%s", imageName, tag)

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

func kubeRestartHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	// WORKAROUND: This is a workaround for `kubectl rollout restart {deployment}`
	// in old version kubernetes cluster.
	// https://github.com/kubernetes/kubernetes/issues/13488

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

		// add or update env name
		found := false
		lastRestart := time.Now().UTC().Format(time.RFC3339)
		for i, env := range result.Spec.Template.Spec.Containers[0].Env {
			if env.Name == LastRestartEnv {
				found = true
				result.Spec.Template.Spec.Containers[0].Env[i].Value = lastRestart
				break
			}
		}
		if !found {
			result.Spec.Template.Spec.Containers[0].Env = append(
				result.Spec.Template.Spec.Containers[0].Env,
				corev1.EnvVar{Name: LastRestartEnv, Value: lastRestart},
			)
		}
		var updateErr error
		dp, updateErr = deploymentsClient.Update(result)
		return updateErr
	})
	if retryErr == errDeploymentIsNotManaged {
		abortWithError(c, http.StatusForbidden, retryErr.Error())
		return
	}
	if retryErr != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment restart error: %v", retryErr))
		return
	}

	// trigger events
	triggerDeploymentEvent(deployment, "restart")

	succeed(c, gin.H{"deployment": getDeployment(dp), "ok": 1})
}

func kubeSetReplicasHandler(c *gin.Context) {
	deployment := c.Param("deployment")

	replicasStr := c.PostForm("replicas")
	replicas, err := strconv.Atoi(replicasStr)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, "params error")
		return
	}

	var newInt32 = func(v int) *int32 {
		var a int32
		a = int32(v)
		return &a
	}

	var dp *appsv1.Deployment
	var hpa *v2beta2.HorizontalPodAutoscaler
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

		var updateErr error

		hpaResult, getErr := hpasClient.Get(deployment, metav1.GetOptions{})
		if getErr != nil && apierrors.IsNotFound(getErr) {
			hpaResult = nil
			getErr = nil
		}
		if getErr != nil {
			return getErr
		}
		if hpaResult != nil {
			hpaResult.Spec.MinReplicas = newInt32(replicas)
			if hpaResult.Spec.MaxReplicas < int32(replicas) {
				hpaResult.Spec.MaxReplicas = int32(replicas)
			}
			hpa, updateErr = hpasClient.Update(hpaResult)
			if updateErr != nil {
				return updateErr
			}
		}

		result.Spec.Replicas = newInt32(replicas)
		dp, updateErr = deploymentsClient.Update(result)
		if updateErr != nil {
			return updateErr
		}
		return nil
	})
	if retryErr == errDeploymentIsNotManaged {
		abortWithError(c, http.StatusForbidden, retryErr.Error())
		return
	}
	if retryErr != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("deployment set-replicas error: %v", retryErr))
		return
	}

	// trigger events
	triggerDeploymentEvent(deployment, "setreplicas")

	d := getDeployment(dp)
	h := getHPA(hpa)

	succeed(c, gin.H{"deployment": d, "hpa": h, "ok": 1})
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

	// get registry bind from labels
	imageBind := getRegistryBind(dp)
	tags, err := registryClient.ListTags(imageBind)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("registry list tags error: %v", err))
		return
	}

	succeed(c, gin.H{"image_name": tags.Name, "tags": tags.Tags})
}

func kubePatchHandler(c *gin.Context) {
	nodesClient := clientset.CoreV1().Nodes()

	var req struct {
		Type         string          `json:"type"`
		ResourceType string          `json:"resource_type"`
		ResourceName string          `json:"resource_name"`
		Payload      json.RawMessage `json:"payload"`
	}
	err := c.BindJSON(&req)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, "params error")
		return
	}
	if req.Type != "patch" && req.Type != "label" {
		abortWithError(c, http.StatusBadRequest, "params error: unsupport type")
		return
	}
	if req.ResourceType != "node" {
		abortWithError(c, http.StatusBadRequest, "params error: unsupport resource type")
		return
	}

	node, err := nodesClient.Get(req.ResourceName, metav1.GetOptions{})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if node == nil {
		abortWithError(c, http.StatusNotFound, "specified resource not found")
		return
	}

	var newNode *corev1.Node
	if req.Type == "patch" {
		newNode, err = nodesClient.Patch(req.ResourceName, types.StrategicMergePatchType, req.Payload)
	} else if req.Type == "label" {
		var operatorData map[string]interface{}
		err = json.Unmarshal(req.Payload, &operatorData)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, "params error: invalid payload")
			return
		}

		var payloads []interface{}

		for key, value := range operatorData {
			payload := PatchStringValue{
				Op:    "add",
				Path:  "/metadata/labels/" + key,
				Value: value,
			}
			payloads = append(payloads, payload)
		}

		payloadBytes, _ := json.Marshal(payloads)
		newNode, err = nodesClient.Patch(req.ResourceName, types.JSONPatchType, payloadBytes)
	}
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	succeed(c, newNode)
}

func kubeNewNodeHandler(c *gin.Context) {
	nodesClient := clientset.CoreV1().Nodes()

	if config.Conf.Kubernetes.NodeNameFormat == "" {
		abortWithError(c, http.StatusInternalServerError, "empty config node_name_format")
		return
	}

	nodeNameCache.lock.Lock()
	defer nodeNameCache.lock.Unlock()

	now := time.Now()
	if nodeNameCache.names == nil || nodeNameCache.lastTime.IsZero() || now.Add(-time.Minute*10).After(nodeNameCache.lastTime) {
		nodeNameCache.names = make(map[string]struct{})
		nodeNameCache.lastTime = now
	}
	defer func() {
		nodeNameCache.lastTime = time.Now()
	}()

	nodeList, err := nodesClient.List(metav1.ListOptions{})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	for _, item := range nodeList.Items {
		nodeNameCache.names[item.Name] = struct{}{}
	}

	var nodeName string
	// NOTICE: max 1000 nodes
	for i := config.Conf.Kubernetes.NodeNameRange[0]; i <= config.Conf.Kubernetes.NodeNameRange[1]; i++ {
		nodeName = fmt.Sprintf(config.Conf.Kubernetes.NodeNameFormat, i)
		if _, ok := nodeNameCache.names[nodeName]; !ok {
			nodeNameCache.names[nodeName] = struct{}{}
			clientIP := c.ClientIP()
			err2 := cloudprovider.SetNodeName(clientIP, nodeName)
			if err2 != nil {
				log.LogError.Errorf("cloud provider set %q node name %q error: %v", clientIP, nodeName, err2)
				// PASS
			}
			succeed(c, map[string]interface{}{
				"node": map[string]interface{}{"name": nodeName},
			})
			return
		}
	}

	abortWithError(c, http.StatusInternalServerError, "no enough node name in pool")
}

func webhookKubeValidateHandler(c *gin.Context) {
	var review admissionv1beta1.AdmissionReview
	err := c.BindJSON(&review)
	if err != nil || review.Request == nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	resp := &admissionv1beta1.AdmissionResponse{
		UID:     review.Request.UID,
		Allowed: false,
		Result: &metav1.Status{
			Status:  "Failure",
			Message: "always deny",
			Reason:  "always deny",
			Code:    402,
		},
	}

	review.Request = nil
	review.Response = resp
	c.JSON(http.StatusOK, &review)
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

	go func() {
		err := webhookClient.Send(event)
		if err != nil {
			log.LogError.Errorf("webhook send deployment %s event error: %v", deployment, err)
		}
	}()
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
		timeoutDuration := 2 * time.Minute
		timeoutCh := time.After(timeoutDuration)
		var lastEvent *DeploymentEvent
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
			event := &DeploymentEvent{
				Name:   deployment,
				Action: action,
				Event:  "processing",
				Status: &dp.Status,
			}
			if isDeploymentComplete(dp, &dp.Status) {
				event.Event = "complete"
				log.LogAccess.Infof("%s deployment %s completed", action, deployment)
			} else if timeout {
				event.Event = "timeout"
				log.LogAccess.Warnf("%s deployment %s timeout", action, deployment)
			}
			if !event.Equal(lastEvent) {
				publishEventTo(deployment, event)
				lastEvent = event
				// reset timeout
				timeoutCh = time.After(timeoutDuration)
			}

			if event.Event != "processing" {
				break
			}
		}
	}()
}

const (
	heartbeat string = "❤️"
)

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
		t, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.LogError.Errorf("Unexpected close error: %v", err)
			}
			break
		}
		if t == websocket.TextMessage || t == websocket.BinaryMessage {
			if heartbeat == string(msg) {
				// heartbeat
				_ = conn.WriteMessage(websocket.TextMessage, []byte(heartbeat))
				continue
			}
		}
	}
}
