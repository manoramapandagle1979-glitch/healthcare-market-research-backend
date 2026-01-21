package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/config"
)

// CloudflareImagesService handles interactions with Cloudflare Images API
type CloudflareImagesService interface {
	Upload(file *multipart.FileHeader, metadata map[string]string) (imageURL string, err error)
	Delete(imageURL string) error
	ExtractImageID(imageURL string) (string, error)
}

type cloudflareImagesService struct {
	config     *config.CloudflareConfig
	httpClient *http.Client
}

// NewCloudflareImagesService creates a new instance of CloudflareImagesService
func NewCloudflareImagesService(cfg *config.CloudflareConfig) CloudflareImagesService {
	return &cloudflareImagesService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CloudflareUploadResponse represents the response from Cloudflare Images API
type CloudflareUploadResponse struct {
	Success bool                    `json:"success"`
	Errors  []CloudflareError       `json:"errors"`
	Result  CloudflareImageResult   `json:"result"`
}

type CloudflareImageResult struct {
	ID       string   `json:"id"`
	Filename string   `json:"filename"`
	Variants []string `json:"variants"`
}

type CloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CloudflareDeleteResponse represents the response from Cloudflare Images delete API
type CloudflareDeleteResponse struct {
	Success bool              `json:"success"`
	Errors  []CloudflareError `json:"errors"`
}

// Upload uploads an image to Cloudflare Images
func (s *cloudflareImagesService) Upload(file *multipart.FileHeader, metadata map[string]string) (string, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Create a buffer to write our multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add the file
	part, err := writer.CreateFormFile("file", file.Filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Add metadata if provided
	if metadata != nil && len(metadata) > 0 {
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return "", fmt.Errorf("failed to marshal metadata: %w", err)
		}
		if err := writer.WriteField("metadata", string(metadataJSON)); err != nil {
			return "", fmt.Errorf("failed to write metadata field: %w", err)
		}
	}

	// Close the writer to finalize the multipart message
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Create the request
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1", s.config.AccountID)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIToken))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the response
	var uploadResp CloudflareUploadResponse
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in response
	if !uploadResp.Success || len(uploadResp.Errors) > 0 {
		if len(uploadResp.Errors) > 0 {
			return "", fmt.Errorf("cloudflare API error: %s", uploadResp.Errors[0].Message)
		}
		return "", fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	// Construct the public URL
	imageURL := fmt.Sprintf("%s/%s/public", s.config.DeliveryURL, uploadResp.Result.ID)
	return imageURL, nil
}

// Delete deletes an image from Cloudflare Images
func (s *cloudflareImagesService) Delete(imageURL string) error {
	// Extract image ID from URL
	imageID, err := s.ExtractImageID(imageURL)
	if err != nil {
		return fmt.Errorf("failed to extract image ID: %w", err)
	}

	// Create the delete request
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1/%s", s.config.AccountID, imageID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIToken))

	// Execute the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the response
	var deleteResp CloudflareDeleteResponse
	if err := json.Unmarshal(respBody, &deleteResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in response
	if !deleteResp.Success || len(deleteResp.Errors) > 0 {
		if len(deleteResp.Errors) > 0 {
			return fmt.Errorf("cloudflare API error: %s", deleteResp.Errors[0].Message)
		}
		return fmt.Errorf("delete failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ExtractImageID extracts the Cloudflare image ID from a full image URL
// Example URL: https://imagedelivery.net/Gy9qXOTaFeYdmWJ69whXhw/2cdc28f0-017a-49c4-9ed7-87056c83901/public
// Returns: 2cdc28f0-017a-49c4-9ed7-87056c83901
func (s *cloudflareImagesService) ExtractImageID(imageURL string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("image URL is empty")
	}

	// Remove the delivery URL prefix
	if !strings.HasPrefix(imageURL, s.config.DeliveryURL) {
		return "", fmt.Errorf("invalid Cloudflare image URL: does not match delivery URL")
	}

	// Remove the delivery URL and leading slash
	remainder := strings.TrimPrefix(imageURL, s.config.DeliveryURL)
	remainder = strings.TrimPrefix(remainder, "/")

	// Split by "/" and get the image ID (second part)
	parts := strings.Split(remainder, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid Cloudflare image URL format")
	}

	imageID := parts[0]
	if imageID == "" {
		return "", fmt.Errorf("image ID is empty in URL")
	}

	return imageID, nil
}
