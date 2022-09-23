package kube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/secrets/vault"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/avast/retry-go/v4"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

const (
	maxRetries          = 5
	delay               = 500 * time.Millisecond
	maxJitter           = 50 * time.Millisecond
	defaultWatchTimeout = int64(60 * 60)
	requestToLimitRatio = 7.0 / 8
)

// empty context.
var noContext = context.Background()

// kubernetesEngine implements a Kubernetes execution engine.
type kubernetesEngine struct {
	client                  *kubernetes.Clientset
	azureStore              core.AzureBlob
	vaultStore              core.Vault
	logger                  lumber.Logger
	cfg                     *config.Config
	spawnedPodsMap          core.SyncMap
	defaultStorageClass     string
	defaultLocalhostProfile string
	defaultImagePullPolicy  v1.PullPolicy
}

// New returns a new k8s engine.
func New(azureStore core.AzureBlob, vaultStore core.Vault, logger lumber.Logger, cfg *config.Config,
	spawnedPodsMap core.SyncMap) (core.K8sRunner, error) {
	k8config, err := newInCluster()
	if err != nil {
		return nil, err
	}
	scName, ok := constants.StorageClassName[cfg.Env]
	if !ok {
		return nil, errs.ErrInvalidEnvironemt
	}
	imagePullPolicy, ok := constants.NucleusImagePolicy[cfg.Env]
	if !ok {
		imagePullPolicy = v1.PullAlways
	}
	k8config.defaultImagePullPolicy = imagePullPolicy
	k8config.logger = logger
	k8config.cfg = cfg
	k8config.vaultStore = vaultStore
	k8config.azureStore = azureStore
	k8config.defaultStorageClass = scName
	k8config.defaultLocalhostProfile = "chrome.json"
	k8config.spawnedPodsMap = spawnedPodsMap
	return k8config, nil
}

// newInCluster returns a new in-cluster engine.
func newInCluster() (*kubernetesEngine, error) {
	// creates the in-cluster config
	k8config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(k8config)
	if err != nil {
		return nil, err
	}
	return &kubernetesEngine{
		client: clientset,
	}, nil
}

func (k *kubernetesEngine) applyResourceQuota(ctx context.Context, namespace string, resourceQuotaCfg *v1.ResourceQuota) error {
	if err := k.withRetry(ctx, "failed to apply resource quota", func() error {
		if _, err := k.client.CoreV1().ResourceQuotas(namespace).Get(ctx, resourceQuotaName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			_, err = k.client.CoreV1().ResourceQuotas(namespace).Create(ctx, resourceQuotaCfg, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			k.logger.Debugf("ResourceQuota %s created successfully on namespace %s", resourceQuotaName, namespace)
			return nil
		}
		if _, err := k.client.CoreV1().ResourceQuotas(namespace).Update(ctx, resourceQuotaCfg, metav1.UpdateOptions{}); err != nil {
			return err
		}
		k.logger.Debugf("ResourceQuota %s updated successfully on namespace %s", resourceQuotaName, namespace)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (k *kubernetesEngine) CreateNamespace(ctx context.Context, license *core.License) error {
	namespace := k.getNamespaceConfig(license)
	namespaceStr := utils.GetRunnerNamespaceFromOrgID(license.OrgID)
	resourceQuotaCfg := k.getResourceQuotaConfig(license)
	if err := k.withRetry(ctx, "failed to create namespace", func() error {
		if _, err := k.client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			k.logger.Debugf("namespace with name %s already exists", namespaceStr)
			return nil
		}
		k.logger.Debugf("namespace created with name: %s", namespaceStr)
		return nil
	}); err != nil {
		return err
	}
	if err := k.applyResourceQuota(ctx, namespaceStr, resourceQuotaCfg); err != nil {
		return err
	}
	return nil
}

func (k *kubernetesEngine) CreateSecret(ctx context.Context, namespace, secretName string,
	secretKey core.UserSecretType, data []byte) error {
	secret := k.getSecret(secretName, string(secretKey), data)
	if err := k.withRetry(ctx, "failed to create secret", func() error {
		if _, err := k.client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			return &errs.ErrSkipRetry{Err: err}
		}
		return nil
	}); err != nil {
		return err
	}
	k.logger.Debugf("secret created with name: %s in namespace %s", secretName, namespace)
	return nil
}

func (k *kubernetesEngine) UpdateSecret(ctx context.Context, namespace, secretName string,
	secretKey core.UserSecretType, data []byte) error {
	secret := k.getSecret(secretName, string(secretKey), data)
	if err := k.withRetry(ctx, "failed to update secret", func() error {
		if _, err := k.client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			if !(apierrors.IsNotFound(err) || apierrors.IsConflict(err)) {
				return err
			}
			return &errs.ErrSkipRetry{Err: err}
		}
		return nil
	}); err != nil {
		return err
	}
	k.logger.Debugf("secret updated with name: %s in namespace %s", secretName, namespace)
	return nil
}

func (k *kubernetesEngine) DeleteSecret(ctx context.Context, namespace, secretName string) error {
	if err := k.withRetry(ctx, "failed to delete secret", func() error {
		if err := k.client.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return &errs.ErrSkipRetry{Err: err}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (k *kubernetesEngine) CreatePersistentVolumeClaim(ctx context.Context, r *core.RunnerOptions, license *core.License) error {
	volumeClaimConfig := k.getPVCConfig(r.PersistentVolumeClaimName, license.Tier)
	if err := k.withRetry(ctx, "failed to create pvc", func() error {
		if _, err := k.client.CoreV1().PersistentVolumeClaims(r.NameSpace).Create(ctx, volumeClaimConfig, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			return &errs.ErrSkipRetry{Err: err}
		}
		return nil
	}); err != nil {
		return err
	}
	k.logger.Debugf("pvc created with name: %s in namespace %s", r.PersistentVolumeClaimName, r.NameSpace)
	return nil
}

func (k *kubernetesEngine) DeletePersistentVolumeClaim(ctx context.Context, r *core.RunnerOptions) error {
	k.logger.Debugf("Deleting pvc with name %s, namespace %s", r.PersistentVolumeClaimName, r.NameSpace)
	if err := k.withRetry(ctx, "failed to delete pvc", func() error {
		if err := k.client.CoreV1().PersistentVolumeClaims(r.NameSpace).Delete(ctx,
			r.PersistentVolumeClaimName, metav1.DeleteOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			return &errs.ErrSkipRetry{Err: err}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

//TODO: explore jobs
// https://github.com/drone-runners/drone-runner-kube/tree/master/engine/podwatcher

// create sends create pod request to k8 api server
func (k *kubernetesEngine) create(ctx context.Context, r *core.RunnerOptions) error {
	pod := k.getPodConfig(r)
	if err := k.withRetry(ctx, "failed to create pod", func() error {
		if _, err := k.client.CoreV1().Pods(r.NameSpace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			return &errs.ErrSkipRetry{Err: err}
		}
		return nil
	}); err != nil {
		return err
	}
	k.logger.Debugf("pod created with name: %s", r.PodName)
	return nil
}

func (k *kubernetesEngine) Destroy(ctx context.Context, r *core.RunnerOptions) error {
	if err := k.withRetry(ctx, "failed to delete pod", func() error {
		err := k.client.CoreV1().Pods(r.NameSpace).Delete(ctx, r.PodName, metav1.DeleteOptions{})
		if !apierrors.IsNotFound(err) {
			return err
		}
		return &errs.ErrSkipRetry{Err: err}
	}); err != nil {
		return err
	}
	return nil
}

func (k *kubernetesEngine) Run(ctx context.Context, r *core.RunnerOptions) error {
	err := k.wait(ctx, r, true, true)
	if err != nil && !errors.Is(err, errs.ErrContainerExited) {
		k.logger.Errorf("error while waiting for running %v", err)
		return err
	}
	// capture logs even if container terminated with non zero status
	strerr := k.streamLogs(ctx, r)
	if strerr != nil {
		k.logger.Errorf("error while streaming logs %v", strerr)
		return strerr
	}

	err = k.wait(ctx, r, false, false)
	if err != nil {
		k.logger.Errorf("error while waiting for terminated %v", err)
		return err
	}
	return nil
}

func (k *kubernetesEngine) Spawn(ctx context.Context, runnerOpts *core.RunnerOptions,
	buildID, taskID string) <-chan error {
	errChan := make(chan error)
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultPodTimeout)
	// store the abort chan in syncmap in order to use it for buildabort service
	key := utils.GetSpawnedPodsMapKey(buildID, taskID)
	abortChan := make(chan struct{})
	k.spawnedPodsMap.Set(key, abortChan)
	go func() {
		<-abortChan
		cancel()
	}()

	go func() {
		defer func() {
			cancel()
			if derr := k.Destroy(noContext, runnerOpts); derr != nil {
				k.logger.Errorf("error while destroying pod: %s, error: %v", runnerOpts.PodName, derr)
			}
		}()
		defer k.spawnedPodsMap.CleanUp(key)
		if err := k.create(ctx, runnerOpts); err != nil {
			k.logger.Errorf("error while creating pod: %s, error: %v", runnerOpts.PodName, err)
			go func() { errChan <- err }()
			return
		}
		if err := k.Run(ctx, runnerOpts); err != nil {
			k.logger.Errorf("error while running pod: %s, error: %v", runnerOpts.PodName, err)
			go func() { errChan <- err }()
			return
		}
		close(errChan)
	}()
	return errChan
}

func (k *kubernetesEngine) WaitForPodTermination(ctx context.Context, podName, podNamespace string) error {
	return k.wait(ctx, &core.RunnerOptions{
		PodName:   podName,
		NameSpace: podNamespace,
	}, false, false)
}

//nolint:gocyclo
func (k *kubernetesEngine) wait(ctx context.Context, r *core.RunnerOptions, waitForInit, waitForOnlyRunning bool) error {
	initPodTerminated := false
	var exitCode int
	watchErr := k.watch(ctx, r, func(e watch.Event) (bool, error) {
		switch t := e.Type; t {
		case watch.Bookmark:
			return false, nil
		case watch.Error:
			pod, ok := e.Object.(*v1.Pod)
			if !ok || pod.ObjectMeta.Name != r.PodName {
				return false, nil
			}
			k.logger.Errorf("error while running pod %s, status: %+v", r.PodName, pod.Status)
			return false, errs.ErrPodStatus

		case watch.Deleted:
			pod, ok := e.Object.(*v1.Pod)
			if !ok || pod.ObjectMeta.Name != r.PodName {
				return false, nil
			}
			k.logger.Debugf("pod %s is deleted, status %+v", r.PodName, pod.Status)
			return false, errs.ErrPodDeleted

		case watch.Added, watch.Modified:
			pod, ok := e.Object.(*v1.Pod)
			if !ok || pod.ObjectMeta.Name != r.PodName {
				return false, nil
			}
			if waitForInit && !initPodTerminated {
				for i := range pod.Status.InitContainerStatuses {
					ics := pod.Status.InitContainerStatuses[i]
					if ics.State.Terminated != nil {
						if ics.State.Terminated.ExitCode != 0 {
							k.logger.Debugf("init container with name %s, terminated with non zero exitcode: %d, message: %s reason: %s",
								ics.Name, ics.State.Terminated.ExitCode, ics.State.Terminated.Message, ics.State.Terminated.Reason)
							return true, errs.ErrInitContainerExited
						}
						initPodTerminated = true
					}
				}
			}
			for i := range pod.Status.ContainerStatuses {
				cs := pod.Status.ContainerStatuses[i]
				if cs.State.Running != nil {
					// terminate the watcher if only we want to wait for container running status
					if waitForOnlyRunning {
						k.logger.Debugf("%s pod is now running", r.PodName)
						return true, nil
					}
					return false, nil
				}
				if cs.State.Terminated != nil {
					exitCode = int(cs.State.Terminated.ExitCode)
					k.logger.Debugf("%s pod terminated with exit code: %d, reason %s", r.PodName, exitCode, cs.State.Terminated.Reason)
					return true, nil
				}
				k.logger.Debugf("%s pod state: %#v", r.PodName, cs.State.Waiting)
				if !(cs.State.Waiting.Reason == "ContainerCreating" || cs.State.Waiting.Reason == "PodInitializing") {
					return false, errors.New(cs.State.Waiting.Reason)
				}
			}
		}
		return false, nil
	})

	if exitCode != 0 {
		if exitCode == constants.OOMKilledExitCode {
			return errs.ErrOOMKilled
		}
		return errs.ErrContainerExited
	}
	return watchErr
}

func (k *kubernetesEngine) watch(ctx context.Context, r *core.RunnerOptions, conditionFunc func(event watch.Event) (bool, error)) error {
	label := fmt.Sprintf("nucleus=%s", r.PodName)
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			// search for specific pod
			return k.client.CoreV1().Pods(r.NameSpace).List(ctx, metav1.ListOptions{
				LabelSelector: label,
			})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return k.client.CoreV1().Pods(r.NameSpace).Watch(ctx, metav1.ListOptions{
				LabelSelector:  label,
				TimeoutSeconds: int64ptr(defaultWatchTimeout),
			})
		},
	}

	preconditionFunc := func(store cache.Store) (bool, error) {
		_, exists, err := store.Get(&metav1.ObjectMeta{Namespace: r.NameSpace, Name: r.PodName})
		if err != nil {
			return true, err
		}
		if !exists {
			return true, err
		}
		return false, nil
	}

	// create an informer, watch till condition is satisfied
	_, err := watchtools.UntilWithSync(ctx, lw, &v1.Pod{}, preconditionFunc, conditionFunc)
	return err
}

func (k *kubernetesEngine) Init(ctx context.Context, r *core.RunnerOptions, payload *core.ParserResponse) error {
	g, errCtx := errgroup.WithContext(ctx)
	orgID := payload.EventBlob.OrgID
	buildID := payload.EventBlob.BuildID
	if payload.SecretPath != "" {
		g.Go(func() error {
			rawSecrets, err := k.vaultStore.ReadSecret(payload.SecretPath)
			if err != nil {
				k.logger.Errorf("failed to read secret in vault, buildID %s, orgID %s, error %v",
					buildID, orgID, err)
				return err
			}
			if len(rawSecrets) != 0 {
				repoSecrets := make(map[string]string, len(rawSecrets))
				for secretName, val := range rawSecrets {
					if repoSecretItem, errP := vault.ParseRepoSecretValue(val); errP == nil {
						repoSecrets[secretName] = repoSecretItem.Value
					}
				}
				secretBody, err := json.Marshal(repoSecrets)
				if err != nil {
					k.logger.Errorf("failed to marshal secret, buildID %s, orgID %s, error %v",
						buildID, orgID, err)
					return err
				}
				err = k.CreateSecret(errCtx, r.NameSpace, utils.GetRepoSecretName(buildID), core.RepoSecret, secretBody)
				if err != nil {
					k.logger.Errorf("failed to create secret in k8s, buildID %s, orgID %s, error %v",
						buildID, orgID, err)
					return err
				}
			}
			return nil
		})
	}
	g.Go(func() error {
		err := k.CreateSecret(errCtx, r.NameSpace, utils.GetTokenSecretName(buildID), core.Oauth, payload.EventBlob.RawToken)
		if err != nil {
			k.logger.Errorf("failed to create secret in k8s, buildID %s, orgID %s, error %v",
				buildID, orgID, err)
			return err
		}
		return nil
	})

	// create pvc for tasks that will be executed.
	g.Go(func() error {
		if err := k.CreatePersistentVolumeClaim(errCtx, r, payload.License); err != nil {
			k.logger.Errorf("failed to create pvc in k8s for buildID %s, orgID %s, error %v",
				buildID, orgID, err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// for exporting logs https://kubernetes.io/docs/concepts/cluster-administration/logging/#sidecar-container-with-logging-agent
func (k *kubernetesEngine) streamLogs(ctx context.Context, r *core.RunnerOptions) error {
	req := k.client.CoreV1().Pods(r.NameSpace).GetLogs(r.PodName, k.getLogOptions(r.ContainerName, true, false))

	reader, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer reader.Close()
	if r.LogfilePath == "" {
		return nil
	}
	return k.cancellableCopy(ctx, r.LogfilePath, reader)
}

// cancellableCopy method copies from source to destination honoring the context.
// If context.Cancel is called, it will return immediately with context canceled error.
func (k *kubernetesEngine) cancellableCopy(ctx context.Context, logFilePath string, src io.ReadCloser) error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		_, err := k.azureStore.UploadStream(ctx, logFilePath, src, core.LogsContainer, gin.MIMEPlain)
		if err != nil {
			k.logger.Errorf("failed to create logfile on azure, error %v ", err)
		}
		ch <- err
	}()

	select {
	case <-ctx.Done():
		src.Close()
		return ctx.Err()
	case err := <-ch:
		return err
	}
}

func (k *kubernetesEngine) Cleanup(ctx context.Context, r *core.RunnerOptions) error {
	if err := k.DeletePersistentVolumeClaim(ctx, r); err != nil {
		k.logger.Errorf("failed to delete pvc in k8s: %v", err)
	}
	if r.Vault != nil {
		if err := k.DeleteSecret(ctx, r.NameSpace, r.Vault.TokenSecretName); err != nil {
			k.logger.Errorf("failed to delete user token in k8s: %v", err)
		}
		if r.Vault.RepoSecretName != "" {
			if err := k.DeleteSecret(ctx, r.NameSpace, r.Vault.RepoSecretName); err != nil {
				k.logger.Errorf("failed to delete user' repo secrets  in k8s: %v", err)
			}
		}
	}
	return nil
}

// wrapper for calling k8s api with exponential backoff retries
func (k *kubernetesEngine) withRetry(ctx context.Context, errMsg string, f func() error) error {
	return retry.Do(func() error {
		return f()
	},
		retry.Context(ctx),
		retry.LastErrorOnly(true),
		retry.Attempts(maxRetries),
		retry.Delay(delay),
		retry.MaxJitter(maxJitter),
		retry.RetryIf(func(err error) bool {
			// skip if error is of type ErrSkipRetry
			var skipErr *errs.ErrSkipRetry
			return !errors.As(err, &skipErr)
		}),
		retry.OnRetry(func(n uint, err error) {
			k.logger.Errorf("%s retry %d, error: %+v", errMsg, n, err)
		}))
}
