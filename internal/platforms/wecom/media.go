package wecom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ruilisi/lsbot/internal/logger"
)

const (
	uploadMediaURL          = "https://qyapi.weixin.qq.com/cgi-bin/media/upload"
	uploadImageURL          = "https://qyapi.weixin.qq.com/cgi-bin/media/uploadimg"
	getMediaURL             = "https://qyapi.weixin.qq.com/cgi-bin/media/get"
	getHDVoiceURL           = "https://qyapi.weixin.qq.com/cgi-bin/media/get/jssdk"
	uploadByURLURL          = "https://qyapi.weixin.qq.com/cgi-bin/media/upload_by_url"
	getUploadByURLResultURL = "https://qyapi.weixin.qq.com/cgi-bin/media/get_upload_by_url_result"
)

// mediaResponse is the common response for media upload APIs.
type mediaResponse struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt string `json:"created_at"`
}

// uploadImageResponse is the response for the upload image API.
type uploadImageResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	URL     string `json:"url"`
}

// uploadByURLResponse is the response for the async upload API.
type uploadByURLResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	JobID   string `json:"jobid"`
}

// UploadByURLStatus represents the status of an async upload job.
type UploadByURLStatus struct {
	Status  int    // 1=processing, 2=done, 3=failed
	MediaID string // populated when Status==2
	ErrCode int
	ErrMsg  string
}

// UploadMedia uploads a temporary media file and returns its media_id (valid for 3 days).
// mediaType must be one of: "image", "voice", "video", "file".
func (p *Platform) UploadMedia(filePath string, mediaType string) (string, error) {
	logger.Info("[WeCom] UploadMedia: path=%s, type=%s", filePath, mediaType)

	token, err := p.getToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	logger.Trace("[WeCom] UploadMedia: got access token")

	body, contentType, err := buildMultipartBody(filePath)
	if err != nil {
		return "", err
	}
	logger.Trace("[WeCom] UploadMedia: built multipart body, size=%d", body.Len())

	url := fmt.Sprintf("%s?access_token=%s&type=%s", uploadMediaURL, token, mediaType)
	resp, err := http.Post(url, contentType, body)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	logger.Trace("[WeCom] UploadMedia: response status=%s, body=%s", resp.Status, string(respBody))

	var result mediaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w (body: %s)", err, string(respBody))
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("upload API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	logger.Info("[WeCom] Uploaded media: type=%s, media_id=%s", mediaType, result.MediaID)
	return result.MediaID, nil
}

// UploadImage uploads an image and returns a permanent URL (for use in news/article messages).
// The URL is only accessible within WeCom contexts.
func (p *Platform) UploadImage(filePath string) (string, error) {
	token, err := p.getToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	body, contentType, err := buildMultipartBody(filePath)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s?access_token=%s", uploadImageURL, token)
	resp, err := http.Post(url, contentType, body)
	if err != nil {
		return "", fmt.Errorf("upload image request failed: %w", err)
	}
	defer resp.Body.Close()

	var result uploadImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("upload image API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	logger.Info("[WeCom] Uploaded image, url=%s", result.URL)
	return result.URL, nil
}

// GetMedia downloads a temporary media file by media_id and saves it to savePath.
func (p *Platform) GetMedia(mediaID string, savePath string) error {
	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s&media_id=%s", getMediaURL, token, mediaID)
	return downloadFile(url, savePath)
}

// GetHDVoice downloads a high-definition voice file (speex 16K) by media_id.
// This provides better quality than GetMedia for voice messages recorded via JSSDK.
func (p *Platform) GetHDVoice(mediaID string, savePath string) error {
	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s&media_id=%s", getHDVoiceURL, token, mediaID)
	return downloadFile(url, savePath)
}

// UploadMediaByURL starts an async upload of a media file from a URL (supports up to 200MB).
// The fileURL must support HTTP Range requests for chunked downloading.
// Returns a jobID that can be used to poll for the result.
func (p *Platform) UploadMediaByURL(fileURL, filename, mediaType string) (string, error) {
	token, err := p.getToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := map[string]any{
		"scene":    1,
		"type":     mediaType,
		"filename": filename,
		"url":      fileURL,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s", uploadByURLURL, token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("async upload request failed: %w", err)
	}
	defer resp.Body.Close()

	var result uploadByURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("async upload API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	logger.Info("[WeCom] Async upload started: jobid=%s", result.JobID)
	return result.JobID, nil
}

// GetUploadByURLResult checks the status of an async upload job.
// Status: 1=processing, 2=done, 3=failed.
func (p *Platform) GetUploadByURLResult(jobID string) (*UploadByURLStatus, error) {
	token, err := p.getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody, _ := json.Marshal(map[string]string{"jobid": jobID})

	url := fmt.Sprintf("%s?access_token=%s", getUploadByURLResultURL, token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("get upload result request failed: %w", err)
	}
	defer resp.Body.Close()

	var raw struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Status  int    `json:"status"`
		Detail  struct {
			ErrCode   int    `json:"errcode"`
			ErrMsg    string `json:"errmsg"`
			MediaID   string `json:"media_id"`
			CreatedAt string `json:"created_at"`
		} `json:"detail"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if raw.ErrCode != 0 {
		return nil, fmt.Errorf("get upload result API error: %d - %s", raw.ErrCode, raw.ErrMsg)
	}

	return &UploadByURLStatus{
		Status:  raw.Status,
		MediaID: raw.Detail.MediaID,
		ErrCode: raw.Detail.ErrCode,
		ErrMsg:  raw.Detail.ErrMsg,
	}, nil
}

// SendMediaMessage sends a media message (file/image/voice/video) to a user.
func (p *Platform) SendMediaMessage(userID, mediaID, mediaType string) error {
	logger.Info("[WeCom] SendMediaMessage: user=%s, media_id=%s, type=%s", userID, mediaID, mediaType)

	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	agentID, _ := strconv.Atoi(p.agentID)
	msg := map[string]any{
		"touser":  userID,
		"msgtype": mediaType,
		"agentid": agentID,
		mediaType: map[string]string{
			"media_id": mediaID,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	logger.Trace("[WeCom] SendMediaMessage: request body=%s", string(body))

	url := fmt.Sprintf("%s?access_token=%s", sendMsgURL, token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send media message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	logger.Trace("[WeCom] SendMediaMessage: response status=%s, body=%s", resp.Status, string(respBody))

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w (body: %s)", err, string(respBody))
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("send media API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	logger.Info("[WeCom] Sent %s message to %s, media_id=%s", mediaType, userID, mediaID)
	return nil
}

// buildMultipartBody creates a multipart/form-data body with the file in a "media" field.
func buildMultipartBody(filePath string) (*bytes.Buffer, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, "", fmt.Errorf("failed to copy file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return body, writer.FormDataContentType(), nil
}

// downloadFile downloads a file from a URL and saves it to savePath.
func downloadFile(url string, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response is an error JSON instead of binary data
	contentType := resp.Header.Get("Content-Type")
	if contentType == "application/json" || contentType == "text/plain" {
		var errResult struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResult); err == nil && errResult.ErrCode != 0 {
			return fmt.Errorf("download API error: %d - %s", errResult.ErrCode, errResult.ErrMsg)
		}
		return fmt.Errorf("unexpected response content-type: %s", contentType)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", savePath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("[WeCom] Downloaded media to %s", savePath)
	return nil
}
