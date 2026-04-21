package storage

import (
	"net/url"
	"testing"
)

func TestParsePublicBaseURL(t *testing.T) {
	t.Parallel()

	u, err := parsePublicBaseURL("https://cdn.example.com")
	if err != nil {
		t.Fatalf("parsePublicBaseURL() error = %v", err)
	}
	if u == nil || u.Scheme != "https" || u.Host != "cdn.example.com" {
		t.Fatalf("unexpected url: %+v", u)
	}
}

func TestParsePublicBaseURLEmpty(t *testing.T) {
	t.Parallel()

	u, err := parsePublicBaseURL("")
	if err != nil {
		t.Fatalf("parsePublicBaseURL() error = %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil url, got %+v", u)
	}
}

func TestPublicURLRewritesSchemeAndHost(t *testing.T) {
	t.Parallel()

	presignedURL, err := url.Parse("http://chat-minio:9000/chat-media/avatar/acc-1?X-Amz-Signature=abc")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	publicBaseURL, err := parsePublicBaseURL("https://files.example.com")
	if err != nil {
		t.Fatalf("parsePublicBaseURL() error = %v", err)
	}

	s := &minioStorage{publicBaseURL: publicBaseURL}
	got := s.publicURL(presignedURL)
	want := "https://files.example.com/chat-media/avatar/acc-1?X-Amz-Signature=abc"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestPublicURLLeavesOriginalWhenPublicBaseURLMissing(t *testing.T) {
	t.Parallel()

	presignedURL, err := url.Parse("http://chat-minio:9000/chat-media/avatar/acc-1?X-Amz-Signature=abc")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	s := &minioStorage{}
	got := s.publicURL(presignedURL)
	if got != presignedURL.String() {
		t.Fatalf("expected original url, got %s", got)
	}
}
