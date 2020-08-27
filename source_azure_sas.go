package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

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
	return r.Method == http.MethodPost && isAzureSASToken(r)
}

func isAzureSASToken(r *http.Request) bool {
	return r.URL.Query().Get("azureSASToken") == "true"
}

type AzureSasRequest struct {
	SASToken       string `json:"sasToken"`
	AccountName    string `json:"accountName"`
	Container      string `json:"container"`
	ImageKey       string `json:"imageKey"`
	OutputImageKey string `json:"outputImageKey"`
}

func (s *AzureSASImageSource) GetImage(r *http.Request) ([]byte, error) {

	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("azure_sas: error reading body: %w", err)
	}
	defer r.Body.Close()

	var azureRequest AzureSasRequest
	if err := json.Unmarshal(d, &azureRequest); err != nil {
		return nil, fmt.Errorf("azure_sas: error parsing request to json: %w", err)
	}

	url, err := assebleBlobURL(
		azureRequest.SASToken,
		azureRequest.AccountName,
		azureRequest.Container,
		azureRequest.ImageKey,
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

	// Injecting the old body so that we can read it twice...
	r.Body = ioutil.NopCloser(bytes.NewBuffer(d))
	return data.Bytes(), nil
}


func uploadBufferToAzureSAS(data []byte, r *http.Request) error {
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("azure_sas: error reading body: %w", err)
	}
	defer r.Body.Close()

	var azureRequest AzureSasRequest
	if err := json.Unmarshal(d, &azureRequest); err != nil {
		return fmt.Errorf("azure_sas: error parsing request to json: %w", err)
	}

	url, err := assebleBlobURL(
		azureRequest.SASToken,
		azureRequest.AccountName,
		azureRequest.Container,
		azureRequest.OutputImageKey,
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
