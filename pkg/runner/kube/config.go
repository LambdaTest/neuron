package kube

import (
	"fmt"
	"strings"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	resourceQuotaName = "org-namespace-resource-quota"
)

const workloadStr = "workload"

type workloadType string

const (
	buildWorkload workloadType = "build"
)

func (k *kubernetesEngine) getResourceQuotaConfig(lic *core.License) *v1.ResourceQuota {
	concurrency := lic.Concurrency
	licenseTierConfig := core.LicenseTierConfigMapping[lic.Tier]
	maxCPU := licenseTierConfig.CPUs
	maxMem := licenseTierConfig.RAM

	return &v1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceQuotaName,
		},
		Spec: v1.ResourceQuotaSpec{
			Hard: v1.ResourceList{
				v1.ResourceRequestsCPU:    computeCPUResource(maxCPU, concurrency),
				v1.ResourceLimitsCPU:      computeCPUResource(maxCPU, concurrency),
				v1.ResourceRequestsMemory: computeMemoryResource(maxMem, concurrency),
				v1.ResourceLimitsMemory:   computeMemoryResource(maxMem, concurrency),
			},
		},
	}
}

func (k *kubernetesEngine) getNamespaceConfig(license *core.License) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: utils.GetRunnerNamespaceFromOrgID(license.OrgID),
		},
	}
}

func (k *kubernetesEngine) getPodConfig(r *core.RunnerOptions) *v1.Pod {
	var tolerations []v1.Toleration
	var affinity *v1.Affinity
	var nodeSelectorMap map[string]string

	if k.cfg.Env != constants.Dev {
		tolerations = []v1.Toleration{
			{
				Key:      workloadStr,
				Operator: v1.TolerationOpEqual,
				Value:    string(buildWorkload),
				Effect:   v1.TaintEffectNoSchedule,
			},
		}
		affinity = &v1.Affinity{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{{
						MatchExpressions: []v1.NodeSelectorRequirement{{
							Key:      workloadStr,
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{string(buildWorkload)},
						}},
					}},
				},
			},
		}
		nodeSelectorMap = map[string]string{workloadStr: string(buildWorkload)}
	}
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.PodName,
			Namespace: r.NameSpace,
			Labels:    map[string]string{"nucleus": r.PodName},
		},
		Spec: v1.PodSpec{
			RestartPolicy:                v1.RestartPolicyNever,
			AutomountServiceAccountToken: boolptr(false),
			Containers:                   []v1.Container{k.getContainerConfig(r)},
			Volumes:                      k.getPodVolumeConfig(r),
			Tolerations:                  tolerations,
			Affinity:                     affinity,
			NodeSelector:                 nodeSelectorMap,
		},
	}
}

func (k *kubernetesEngine) getContainerConfig(r *core.RunnerOptions) v1.Container {
	c := v1.Container{
		Name:            r.ContainerName,
		Image:           r.DockerImage,
		Args:            r.ContainerArgs,
		ImagePullPolicy: k.defaultImagePullPolicy,
		Ports:           []v1.ContainerPort{{ContainerPort: int32(r.ContainerPort)}},
		Resources:       k.getResources(r.Tier),
		VolumeMounts:    k.getContainerVolumeConfig(r),
		Env:             k.getEnvVariables(r),
		SecurityContext: &v1.SecurityContext{
			SeccompProfile: &v1.SeccompProfile{
				Type: v1.SeccompProfileTypeLocalhost, LocalhostProfile: stringptr(k.defaultLocalhostProfile),
			},
			AllowPrivilegeEscalation: boolptr(false),
		},
	}

	return c
}

func (k *kubernetesEngine) getPodVolumeConfig(r *core.RunnerOptions) []v1.Volume {
	if r.PersistentVolumeClaimName == "" {
		return nil
	}
	var volumes []v1.Volume
	volumes = append(volumes, v1.Volume{
		Name: constants.DefaultVolumeName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: r.PersistentVolumeClaimName,
			},
		},
	})
	if r.Vault == nil {
		return volumes
	}
	var volumeProjections []v1.VolumeProjection
	if r.Vault.RepoSecretName != "" {
		volumeProjections = append(volumeProjections, v1.VolumeProjection{
			Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: r.Vault.RepoSecretName,
				},
				Optional: boolptr(false),
			},
		})
	}
	if r.Vault.TokenSecretName != "" {
		volumeProjections = append(volumeProjections, v1.VolumeProjection{
			Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: r.Vault.TokenSecretName,
				},
				Optional: boolptr(false),
			},
		})
	}
	if len(volumeProjections) > 0 {
		volumes = append(volumes, v1.Volume{
			Name: constants.DefaultSecResourceName,
			VolumeSource: v1.VolumeSource{
				Projected: &v1.ProjectedVolumeSource{
					Sources: volumeProjections,
				},
			},
		})
	}
	return volumes
}

func (k *kubernetesEngine) getContainerVolumeConfig(r *core.RunnerOptions) []v1.VolumeMount {
	var mounts []v1.VolumeMount
	if r.Vault != nil && r.Vault.TokenSecretName != "" {
		mounts = append(mounts, v1.VolumeMount{
			Name:      constants.DefaultSecResourceName,
			MountPath: constants.DefaultVaultVolumePath,
			ReadOnly:  true,
		})
	}
	if r.PersistentVolumeClaimName != "" {
		mounts = append(mounts, v1.VolumeMount{
			Name:      constants.DefaultVolumeName,
			MountPath: constants.DefaultContainerVolumePath,
		})
	}
	return mounts
}

func (k *kubernetesEngine) getPVCConfig(claimName string, licenseTier core.Tier) *v1.PersistentVolumeClaim {
	storageRequest := resource.MustParse(core.LicenseTierConfigMapping[licenseTier].Storage)
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: claimName,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			StorageClassName: stringptr(k.defaultStorageClass),
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: storageRequest,
				},
			},
		},
	}
}
func (k *kubernetesEngine) getResources(tier core.Tier) v1.ResourceRequirements {
	var dst v1.ResourceRequirements

	tierConfig := core.TaskTierConfigMapping[tier]
	cpuLimit := resource.MustParse(tierConfig.CPUs)
	memLimit := resource.MustParse(tierConfig.RAM)
	// reducing requests by `requestToLimitRatio`
	cpuRequest := resource.MustParse(fmt.Sprintf("%f", requestToLimitRatio*cpuLimit.AsApproximateFloat64()))
	memRequest := resource.MustParse(fmt.Sprintf("%f", requestToLimitRatio*memLimit.AsApproximateFloat64()))

	dst.Limits = v1.ResourceList{}
	dst.Limits[v1.ResourceCPU] = cpuLimit
	dst.Limits[v1.ResourceMemory] = memLimit

	dst.Requests = v1.ResourceList{}
	dst.Requests[v1.ResourceCPU] = cpuRequest
	dst.Requests[v1.ResourceMemory] = memRequest

	return dst
}

func (k *kubernetesEngine) getEnvVariables(r *core.RunnerOptions) []v1.EnvVar {
	var envVars []v1.EnvVar
	//TODO: can mount as secrets
	if r.PodType == core.CoveragePod {
		envVars = append(envVars, v1.EnvVar{
			Name:  "AZURE_STORAGE_ACCOUNT",
			Value: k.cfg.Azure.StorageAccountName,
		}, v1.EnvVar{
			Name:  "AZURE_STORAGE_ACCESS_KEY",
			Value: k.cfg.Azure.StorageAccessKey,
		}, v1.EnvVar{
			Name:  "AZURE_CONTAINER_NAME",
			Value: k.cfg.Azure.CoverageContainerName,
		})
	}
	if len(r.Env) > 0 {
		for _, env := range r.Env {
			i := strings.Index(env, "=")
			envVars = append(envVars, v1.EnvVar{
				Name:  env[:i],
				Value: env[i+1:],
			})
		}
	}

	return envVars
}

func (k *kubernetesEngine) getLogOptions(containerName string, follow, previous bool) *v1.PodLogOptions {
	return &v1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
		Previous:  previous,
	}
}

func (k *kubernetesEngine) getSecret(name, secretKey string, data []byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Type: "Opaque",
		Data: map[string][]byte{
			secretKey: data,
		},
	}
}

func computeMemoryResource(licenseValue string, concurrency int64) resource.Quantity {
	resQuantity := resource.MustParse(licenseValue)
	total := fmt.Sprintf("%d", resQuantity.Value()*concurrency)
	return resource.MustParse(total)
}

func computeCPUResource(licenseValue string, concurrency int64) resource.Quantity {
	resQuantity := resource.MustParse(licenseValue)
	total := fmt.Sprintf("%f", resQuantity.AsApproximateFloat64()*float64(concurrency))
	return resource.MustParse(total)
}

func boolptr(v bool) *bool {
	return &v
}

func stringptr(v string) *string {
	return &v
}

func int64ptr(v int64) *int64 {
	return &v
}
