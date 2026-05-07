package slack

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	goslack "github.com/slack-go/slack"
)

func (s *Service) getMessageImageURL(message goslack.Message) (string, bool) {
	for _, file := range message.Files {
		if !strings.HasPrefix(file.Mimetype, "image/") {
			continue
		}
		hasImage := true
		if file.Mimetype == "image/heic" || file.Mimetype == "image/heif" {
			if localURL := s.downloadSlackHeicAsJpeg(file); localURL != "" {
				return localURL, hasImage
			}
			continue
		}
		if localURL := s.downloadSlackFile(file); localURL != "" {
			return localURL, hasImage
		}
		if file.PermalinkPublic != "" {
			return file.PermalinkPublic, hasImage
		}
		return "", hasImage
	}

	for _, attachment := range message.Attachments {
		if attachment.ImageURL != "" {
			return attachment.ImageURL, true
		}
		if attachment.ThumbURL != "" {
			return attachment.ThumbURL, true
		}
	}

	return "", false
}

func (s *Service) getMessageAttachment(message goslack.Message) (string, string, bool) {
	for _, file := range message.Files {
		if strings.HasPrefix(file.Mimetype, "image/") {
			if file.Mimetype != "image/heic" && file.Mimetype != "image/heif" {
				continue
			}
		}
		if !strings.HasPrefix(file.Mimetype, "video/") && !strings.HasPrefix(file.Mimetype, "audio/") && !strings.HasPrefix(file.Mimetype, "image/") {
			continue
		}
		hasAttachment := true
		if file.Mimetype == "image/heic" || file.Mimetype == "image/heif" {
			if localURL := s.downloadSlackHeicAsJpeg(file); localURL != "" {
				return localURL, "image/jpeg", hasAttachment
			}
		}
		if localURL := s.downloadSlackFile(file); localURL != "" {
			return localURL, file.Mimetype, hasAttachment
		}
		if file.PermalinkPublic != "" {
			return file.PermalinkPublic, file.Mimetype, hasAttachment
		}
		return "", file.Mimetype, hasAttachment
	}

	return "", "", false
}

func (s *Service) downloadSlackFile(file goslack.File) string {
	if s.attachmentsDir == "" || s.attachmentsURL == "" || s.slackToken == "" {
		return ""
	}

	downloadURL := file.URLPrivateDownload
	if downloadURL == "" {
		downloadURL = file.URLPrivate
	}
	if downloadURL == "" {
		return ""
	}

	filename := s.buildAttachmentFilename(file)
	if filename == "" {
		return ""
	}

	localURL, err := s.downloadFile(downloadURL, filename)
	if err != nil {
		return ""
	}

	return localURL
}

func (s *Service) downloadSlackHeicAsJpeg(file goslack.File) string {
	if s.attachmentsDir == "" || s.attachmentsURL == "" || s.slackToken == "" {
		return ""
	}

	downloadURL := file.URLPrivateDownload
	if downloadURL == "" {
		downloadURL = file.URLPrivate
	}
	if downloadURL == "" {
		return ""
	}

	baseName := file.ID
	if baseName == "" {
		return ""
	}

	heicName := baseName + ".heic"
	heicPath := filepath.Join(s.attachmentsDir, heicName)
	jpgName := baseName + ".jpg"
	jpgPath := filepath.Join(s.attachmentsDir, jpgName)

	if _, err := os.Stat(jpgPath); err == nil {
		return s.attachmentsURL + "/" + jpgName
	}

	if _, err := os.Stat(heicPath); err != nil {
		if _, err := s.downloadFile(downloadURL, heicName); err != nil {
			return ""
		}
	}

	if err := convertHeicToJpeg(heicPath, jpgPath); err != nil {
		if s.logger != nil {
			s.logger.Err(err).KV("heic_path", heicPath).KV("jpg_path", jpgPath).Error("failed to convert heic")
		}
		return ""
	}

	return s.attachmentsURL + "/" + jpgName
}

func convertHeicToJpeg(sourcePath, targetPath string) error {
	if _, err := exec.LookPath("heif-convert"); err == nil {
		return exec.Command("heif-convert", sourcePath, targetPath).Run()
	}
	if _, err := exec.LookPath("magick"); err == nil {
		return exec.Command("magick", sourcePath, targetPath).Run()
	}
	if _, err := exec.LookPath("convert"); err == nil {
		return exec.Command("convert", sourcePath, targetPath).Run()
	}
	return fmt.Errorf("no heic converter available")
}

func (s *Service) buildAttachmentFilename(file goslack.File) string {
	ext := sanitizeExtension(file.Filetype)
	if ext == "" {
		ext = "img"
	}
	if file.ID == "" {
		return ""
	}
	return file.ID + "." + ext
}

func sanitizeExtension(ext string) string {
	ext = strings.ToLower(ext)
	for _, r := range ext {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return ""
	}
	return ext
}

func (s *Service) downloadFile(url, filename string) (string, error) {
	if err := os.MkdirAll(s.attachmentsDir, 0o755); err != nil {
		return "", err
	}

	destPath := filepath.Join(s.attachmentsDir, filename)
	if _, err := os.Stat(destPath); err == nil {
		return s.attachmentsURL + "/" + filename, nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.slackToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status downloading file: %s", resp.Status)
	}

	tmpPath := destPath + ".part"
	fileHandle, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer fileHandle.Close()

	if _, err := io.Copy(fileHandle, resp.Body); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	return s.attachmentsURL + "/" + filename, nil
}
