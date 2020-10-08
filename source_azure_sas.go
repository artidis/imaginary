package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

const ImageSourceTypeAzureSAS ImageSourceType = "azure_sas"

func init() {
	RegisterSource(ImageSourceTypeAzureSAS, NewAzureSASImageSource)
}

type AzureSASImageSource struct {
	Config *SourceConfig
}

func NewAzureSASImageSource(config *SourceConfig) ImageSource {
	return &AzureSASImageSource{Config: config}
}

func (s *AzureSASImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodGet && parseAzureSASToken(r) != ""
}

func (s *AzureSASImageSource) GetImage(r *http.Request) ([]byte, error) {
	sasToken := parseAzureSASToken(r)
	accountName := os.Getenv("AZURE_ACCOUNT_NAME")
	container := parseAzureContainer(r)
	imageKey := parseAzureBlobKey(r)

	u, err := assebleBlobURL(
		sasToken,
		accountName,
		container,
		imageKey,
	)
	if err != nil {
		return nil, fmt.Errorf("azure_sas: error parsing url: %w", err)
	}

	blobURL := azblob.NewBlobURL(
		*u,
		azblob.NewPipeline(
			azblob.NewAnonymousCredential(),
			azblob.PipelineOptions{},
		),
	)

	dlResp, err := blobURL.Download(r.Context(), 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, fmt.Errorf("azure_sas: error downloading blob: %w", err)
	}

	data := &bytes.Buffer{}
	bodyData := dlResp.Body(azblob.RetryReaderOptions{})
	defer bodyData.Close()

	if _, err := data.ReadFrom(bodyData); err != nil {
		return nil, fmt.Errorf("azure_sas: error reading data: %w", err)
	}

	return data.Bytes(), nil
}

func uploadBufferToAzureSAS(data []byte, sasURL *url.URL) error {
	blobURL := azblob.NewBlobURL(
		*sasURL,
		azblob.NewPipeline(
			azblob.NewAnonymousCredential(),
			azblob.PipelineOptions{},
		),
	).ToBlockBlobURL()

	if _, err := blobURL.Upload(
		context.Background(),
		bytes.NewReader(data),
		azblob.BlobHTTPHeaders{},
		azblob.Metadata{},
		azblob.BlobAccessConditions{},
	); err != nil {
		return fmt.Errorf("azure_sas: uploading image failed: %w", err)
	}

	return nil
}

func parseAzureSASToken(request *http.Request) string {
	return request.URL.Query().Get("azureSASBlobURL")
}

type AzureSASSource struct {
	SASToken    string
	AccountName string
}

func (a *AzureSASSource) DownloadImage(container, imageKey string) ([]byte, error) {
	url, err := assebleBlobURL(
		a.SASToken,
		a.AccountName,
		container,
		imageKey,
	)
	if err != nil {
		return nil, fmt.Errorf("azure_sas: error assembling url path: %w", err)
	}

	blobURL := azblob.NewBlobURL(
		*url,
		azblob.NewPipeline(
			azblob.NewAnonymousCredential(),
			azblob.PipelineOptions{},
		),
	)

	dlResp, err := blobURL.Download(context.Background(), 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, fmt.Errorf("azure_sas: error downloading blob: %w", err)
	}

	data := &bytes.Buffer{}
	bodyData := dlResp.Body(azblob.RetryReaderOptions{})
	defer bodyData.Close()

	if _, err := data.ReadFrom(bodyData); err != nil {
		return nil, fmt.Errorf("azure_sas: error reading data: %w", err)
	}

	return data.Bytes(), nil
}

func (a *AzureSASSource) UploadImage(data []byte, fileKey, container string) error {
	url, err := assebleBlobURL(
		a.SASToken,
		a.AccountName,
		container,
		fileKey,
	)
	if err != nil {
		return fmt.Errorf("azure_sas: error assembling url path: %w", err)
	}

	blobURL := azblob.NewBlobURL(
		*url,
		azblob.NewPipeline(
			azblob.NewAnonymousCredential(),
			azblob.PipelineOptions{},
		),
	).ToBlockBlobURL()

	if _, err := blobURL.Upload(
		context.Background(),
		bytes.NewReader(data),
		azblob.BlobHTTPHeaders{},
		azblob.Metadata{},
		azblob.BlobAccessConditions{},
	); err != nil {
		return fmt.Errorf("azure_sas: uploading image failed: %w", err)
	}

	return nil
}

func assebleBlobURL(sasToken, accountName, container, blobKey string) (*url.URL, error) {
	return url.ParseRequestURI(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
		accountName, container, blobKey, sasToken))
}
