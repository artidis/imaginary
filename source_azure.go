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

const ImageSourceTypeAzure ImageSourceType = "azure"

func init() {
	RegisterSource(ImageSourceTypeAzure, NewAzureImageSource)
}

func newAzureSession(container string) (*azblob.ContainerURL, error) {
	accountName := os.Getenv("AZURE_ACCOUNT_NAME")
	accountKey := os.Getenv("AZURE_ACCOUNT_KEY")

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("azure: credential error: %w", err)
	}

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", accountName))
	containerURL := azblob.NewServiceURL(*u, p).NewContainerURL(container)

	return &containerURL, nil
}

type AzureImageSource struct {
	Config *SourceConfig
}

func NewAzureImageSource(config *SourceConfig) ImageSource {
	return &AzureImageSource{Config: config}
}

func (s *AzureImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodGet && parseAzureBlobKey(r) != ""
}

func (s *AzureImageSource) GetImage(r *http.Request) ([]byte, error) {
	key, container := parseAzureBlobKey(r), parseAzureContainer(r)

	session, err := newAzureSession(container)
	if err != nil {
		return nil, fmt.Errorf("azure: error getting azure session: %w", err)
	}

	dlResp, _ := session.NewBlobURL(key).
		Download(r.Context(), 0, 0, azblob.BlobAccessConditions{}, false)

	data := &bytes.Buffer{}
	bodyData := dlResp.Body(azblob.RetryReaderOptions{})
	defer bodyData.Close()

	if _, err := data.ReadFrom(bodyData); err != nil {
		return nil, fmt.Errorf("azure: error reading data: %w", err)
	}

	return data.Bytes(), nil
}

func uploadBufferToAzure(data []byte, outputBlobKey, container string) error {
	session, err := newAzureSession(container)
	if err != nil {
		return fmt.Errorf("azure: error getting azure session: %w", err)
	}

	if _, err := session.
		NewBlockBlobURL(outputBlobKey).
		Upload(
			context.Background(),
			bytes.NewReader(data),
			azblob.BlobHTTPHeaders{},
			azblob.Metadata{},
			azblob.BlobAccessConditions{},
		); err != nil {
		return fmt.Errorf("azure: uploading image failed: %w", err)
	}

	return nil
}

func parseAzureBlobKey(request *http.Request) string {
	return request.URL.Query().Get("azureBlobKey")
}

func parseAzureBlobOutputKey(request *http.Request) string {
	return request.URL.Query().Get("azureOutputBlobKey")
}

func parseAzureContainer(request *http.Request) string {
	return request.URL.Query().Get("azureContainer")
}
