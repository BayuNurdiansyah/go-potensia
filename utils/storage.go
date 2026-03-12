package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxAvatarSize = 5 << 20 // 5 MB
	avatarBucket  = "avatars"
)

// AllowedAvatarTypes is the set of accepted MIME types for avatars.
var AllowedAvatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// SupabaseStorage wraps Supabase Storage REST API calls.
type SupabaseStorage struct {
	projectURL string // e.g. https://xxxx.supabase.co
	serviceKey string // service_role key (server-side only)
	httpClient *http.Client
}

// NewSupabaseStorage creates a storage client from environment variables.
func NewSupabaseStorage() *SupabaseStorage {
	return &SupabaseStorage{
		projectURL: strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		serviceKey: os.Getenv("SUPABASE_SERVICE_KEY"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// UploadAvatar uploads a multipart file to Supabase Storage and returns the public URL.
// objectPath format: "mentor/42.jpg" or "parent/7.png"
func (s *SupabaseStorage) UploadAvatar(file multipart.File, header *multipart.FileHeader, folder string, userID uint) (string, error) {
	// 1. Validate size
	if header.Size > maxAvatarSize {
		return "", fmt.Errorf("file terlalu besar, maksimum 5 MB")
	}

	// 2. Detect & validate MIME type
	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return "", fmt.Errorf("gagal membaca file")
	}
	mimeType := http.DetectContentType(buf)
	ext, ok := AllowedAvatarTypes[mimeType]
	if !ok {
		return "", fmt.Errorf("format tidak didukung, gunakan JPEG, PNG, atau WebP")
	}

	// 3. Reset reader to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("gagal memproses file")
	}

	// 4. Build object path: e.g. "mentor/42_1710000000.jpg"
	objectPath := fmt.Sprintf("%s/%d_%d%s", folder, userID, time.Now().Unix(), ext)

	// 5. Read full file into buffer
	body, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file")
	}

	// 6. Upload to Supabase Storage via REST
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.projectURL, avatarBucket, objectPath)
	req, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gagal membuat request upload")
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("x-upsert", "true") // overwrite if exists

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal menghubungi storage: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("storage error %d: %s", resp.StatusCode, string(b))
	}

	// 7. Return public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.projectURL, avatarBucket, objectPath)
	return publicURL, nil
}

// DeleteAvatar deletes a file from Supabase Storage by its full public URL.
// Safe to call with empty string (no-op).
func (s *SupabaseStorage) DeleteAvatar(publicURL string) error {
	if publicURL == "" {
		return nil
	}

	// Extract object path from URL
	// URL format: https://xxx.supabase.co/storage/v1/object/public/avatars/mentor/42_xxx.jpg
	marker := "/object/public/" + avatarBucket + "/"
	idx := strings.Index(publicURL, marker)
	if idx == -1 {
		return nil // not a supabase storage URL — skip
	}
	objectPath := publicURL[idx+len(marker):]

	deleteURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.projectURL, avatarBucket, objectPath)
	req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ValidateAvatarFile checks size and extension before opening.
// Call this before ShouldBind to give early errors.
func ValidateAvatarFile(header *multipart.FileHeader) error {
	if header.Size > maxAvatarSize {
		return fmt.Errorf("file terlalu besar, maksimum 5 MB")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowed[ext] {
		return fmt.Errorf("ekstensi tidak didukung, gunakan .jpg .png .webp")
	}
	return nil
}