package errors

import "github.com/Azure/azure-storage-blob-go/azblob"

// AzureError is an error type that represents an error returned from Azure.
func AzureError(err error) error {
	if err == nil {
		return nil
	}
	if serr, ok := err.(azblob.StorageError); ok { // This error is a Service-specific
		if serr.ServiceCode() == azblob.ServiceCodeBlobNotFound { // Compare serviceCode to ServiceCodeXxx constants
			return ErrNotFound
		}
	}
	return err
}
