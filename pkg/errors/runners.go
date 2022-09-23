package errors

var (
	// ErrPodDeleted is returned when the k8s pod gets deleted.
	ErrPodDeleted = New("Pod Deleted")
	// ErrContainerExited is returned when the contanier gets terminated after non zero exit code.
	ErrContainerExited = New("Pod exited with non zero exit code")
	// ErrOOMKilled is returned when container terminated with OOM Killed error.
	ErrOOMKilled = New("OOM Killed")
	// ErrInitContainerExited is returned when the init contanier gets terminated after non zero exit code.
	ErrInitContainerExited = New("Vault Init container with non zero exit code")
	// ErrPodStatus is returned when the pod has Error status.
	ErrPodStatus = New("Error pod status")
	// ErrInvalidSlug is returned when the slug (orgName/repoName) is invalid.
	ErrInvalidSlug = New("Invalid repo slug")
)
