package slack

import (
	"testing"

	goslack "github.com/slack-go/slack"
)

func TestSanitizeExtension(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "PNG", want: "png"},
		{in: "mp4", want: "mp4"},
		{in: "tar.gz", want: ""},
		{in: "bad-ext!", want: ""},
	}

	for _, tt := range tests {
		if got := sanitizeExtension(tt.in); got != tt.want {
			t.Fatalf("sanitizeExtension(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildAttachmentFilename(t *testing.T) {
	svc := &Service{}

	if got := svc.buildAttachmentFilename(goslack.File{ID: "F123", Filetype: "png"}); got != "F123.png" {
		t.Fatalf("unexpected filename %q", got)
	}
	if got := svc.buildAttachmentFilename(goslack.File{ID: "F123", Filetype: "bad-ext"}); got != "F123.img" {
		t.Fatalf("expected fallback img extension, got %q", got)
	}
	if got := svc.buildAttachmentFilename(goslack.File{Filetype: "png"}); got != "" {
		t.Fatalf("expected empty filename without ID, got %q", got)
	}
}

func TestGetMessageImageURLAndAttachmentFallbacks(t *testing.T) {
	svc := &Service{}
	message := goslack.Message{
		Msg: goslack.Msg{
			Files: []goslack.File{
				{
					Mimetype:        "image/png",
					PermalinkPublic: "https://example.com/image.png",
				},
				{
					Mimetype:        "video/mp4",
					PermalinkPublic: "https://example.com/video.mp4",
				},
			},
		},
	}

	imageURL, hasImage := svc.getMessageImageURL(message)
	if !hasImage || imageURL != "https://example.com/image.png" {
		t.Fatalf("unexpected image result %v %q", hasImage, imageURL)
	}

	attachmentURL, mime, hasAttachment := svc.getMessageAttachment(message)
	if !hasAttachment || attachmentURL != "https://example.com/video.mp4" || mime != "video/mp4" {
		t.Fatalf("unexpected attachment result %v %q %q", hasAttachment, attachmentURL, mime)
	}
}
