package azure

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"

	"github.com/LambdaTest/neuron/pkg/lumber"
)

// store represents the azure storage
type store struct {
	storageAccountURL     string
	cdnURL                string
	sharedKeyCredential   *azblob.SharedKeyCredential
	logger                lumber.Logger
	coverageContainerName string
	metricsContainerName  string
	payloadContainerName  string
	cacheContainerName    string
	logsContainerName     string
	service               azblob.ServiceURL
}

const (
	defaultBufferSize  = 4 * 1024 * 1024
	defaultParallelism = 16
	defaultMaxBuffers  = 4
	sasLinkTimeout     = 12 * time.Hour
)

// NewAzureBlobEnv returns a new Azure blob store.
func NewAzureBlobEnv(cfg *config.Config, logger lumber.Logger) (core.AzureBlob, error) {
	if cfg.Azure.StorageAccountName == "" ||
		cfg.Azure.StorageAccessKey == "" ||
		cfg.Azure.PayloadContainerName == "" ||
		cfg.Azure.CacheContainerName == "" ||
		cfg.Azure.LogsContainerName == "" ||
		cfg.Azure.MetricsContainerName == "" ||
		cfg.Azure.CdnURL == "" ||
		cfg.Azure.CoverageContainerName == "" {
		return nil, errs.ErrAzureConfig
	}
	// Create a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(cfg.Azure.StorageAccountName, cfg.Azure.StorageAccessKey)
	if err != nil {
		logger.Errorf("Invalid azure credentials, error: %v", err)
		return nil, err
	}
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", cfg.Azure.StorageAccountName))
	pipe := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	return &store{
		cdnURL:                cfg.Azure.CdnURL,
		storageAccountURL:     u.String(),
		sharedKeyCredential:   credential,
		cacheContainerName:    cfg.Azure.CacheContainerName,
		metricsContainerName:  cfg.Azure.MetricsContainerName,
		payloadContainerName:  cfg.Azure.PayloadContainerName,
		logsContainerName:     cfg.Azure.LogsContainerName,
		coverageContainerName: cfg.Azure.CoverageContainerName,
		service:               azblob.NewServiceURL(*u, pipe),
		logger:                logger,
	}, nil
}

func (s *store) UploadStream(ctx context.Context, blobPath string, reader io.Reader, containerID int, mimeType string) (string, error) {
	containerName, err := s.getContainerName(containerID)
	if err != nil {
		s.logger.Errorf("failed to find container for id %d, error %v", containerID, err)
		return "", err
	}
	containerURL := s.service.NewContainerURL(containerName)
	blobURL := containerURL.NewBlockBlobURL(blobPath)

	s.logger.Debugf("%s", blobURL.String())
	_, err = azblob.UploadStreamToBlockBlob(ctx, reader, blobURL, azblob.UploadStreamToBlockBlobOptions{
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{ContentType: mimeType},
		BufferSize:      defaultBufferSize,
		MaxBuffers:      defaultMaxBuffers,
	})

	return blobURL.String(), err
}

func (s *store) DownloadStream(ctx context.Context, blobPath string, containerID int) (io.ReadCloser, error) {
	containerName, err := s.getContainerName(containerID)
	if err != nil {
		s.logger.Errorf("failed to find container for id %d, error %v", containerID, err)
		return nil, err
	}
	containerURL := s.service.NewContainerURL(containerName)
	blobURL := containerURL.NewBlockBlobURL(blobPath)

	out, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, errs.AzureError(err)
	}
	return out.Body(azblob.RetryReaderOptions{MaxRetryRequests: 5}), nil
}

func (s *store) UploadBytes(ctx context.Context, blobPath string, rawBytes []byte, containerID int, mimeType string) (string, error) {
	containerName, err := s.getContainerName(containerID)
	if err != nil {
		s.logger.Errorf("failed to find container for id %d, error %v", containerID, err)
		return "", err
	}
	containerURL := s.service.NewContainerURL(containerName)
	blobURL := containerURL.NewBlockBlobURL(blobPath)
	s.logger.Debugf("uploading bytes to blob %s", blobURL.String())
	_, err = azblob.UploadBufferToBlockBlob(ctx, rawBytes, blobURL, azblob.UploadToBlockBlobOptions{
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{ContentType: mimeType},
		BlockSize:       defaultBufferSize,
		Parallelism:     defaultParallelism,
	})

	return blobURL.String(), err
}

func (s *store) GenerateSasURL(ctx context.Context, blobPath string, containerID int) (string, error) {
	containerName, err := s.getContainerName(containerID)
	if err != nil {
		s.logger.Errorf("failed to find container for id %d", containerID)
		return "", err
	}
	containerURL := s.service.NewContainerURL(containerName)

	sasDefaultSignature := azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,
		ExpiryTime:    time.Now().UTC().Add(sasLinkTimeout),
		ContainerName: containerName,
		BlobName:      blobPath,
		Permissions:   azblob.BlobSASPermissions{Read: true, Add: true, Write: true}.String(),
	}
	sasQueryParams, err := sasDefaultSignature.NewSASQueryParameters(s.sharedKeyCredential)
	if err != nil {
		s.logger.Errorf("failed to generated sas query params, error %v", err)
		return "", err
	}

	parts := azblob.BlobURLParts{
		Scheme:        containerURL.URL().Scheme,
		Host:          containerURL.URL().Host,
		ContainerName: containerName,
		BlobName:      blobPath,
		SAS:           sasQueryParams,
	}

	rawURL := parts.URL()
	return rawURL.String(), nil
}
func (s *store) getContainerName(containerID int) (string, error) {
	switch containerID {
	case core.PayloadContainer:
		return s.payloadContainerName, nil
	case core.MetricsContainer:
		return s.metricsContainerName, nil
	case core.CacheContainer:
		return s.cacheContainerName, nil
	case core.LogsContainer:
		return s.logsContainerName, nil
	case core.CoverageContainer:
		return s.coverageContainerName, nil
	default:
		return "", errs.ErrUnknownContainer
	}
}

func (s *store) ReplaceWithCDN(urlStr string) string {
	return strings.Replace(urlStr, s.storageAccountURL, s.cdnURL, 1)
}
