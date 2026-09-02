package auth

import (
	"strings"
	"testing"
	"time"
)

func TestJWTAccessRoundTrip(t *testing.T) {
	j, err := NewJWT([]byte("test-jwt-secret-at-least-32-bytes!!"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	s, err := j.SignAccess("user1", "ses1", now)
	if err != nil {
		t.Fatal(err)
	}
	uid, sid, err := j.ParseAccess(s)
	if err != nil {
		t.Fatal(err)
	}
	if uid != "user1" || sid != "ses1" {
		t.Fatalf("uid=%s sid=%s", uid, sid)
	}
}

func TestJWTPendingNotAccess(t *testing.T) {
	j, err := NewJWT([]byte("test-jwt-secret-at-least-32-bytes!!"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	p, err := j.SignPending("user1", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := j.ParseAccess(p); err == nil {
		t.Fatal("pending erişim olarak kabul edildi")
	}
	uid, err := j.ParsePending(p)
	if err != nil || uid != "user1" {
		t.Fatalf("pending: uid=%s err=%v", uid, err)
	}
}

func TestJWTSecretRequired(t *testing.T) {
	if _, err := NewJWT(nil); err == nil {
		t.Fatal("boş sır kabul edildi")
	}
	if _, err := NewJWT([]byte{}); err == nil {
		t.Fatal("boş sır kabul edildi")
	}
}

func TestJWTAccessOmitsPII(t *testing.T) {
	j, err := NewJWT([]byte("test-jwt-secret-at-least-32-bytes!!"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := j.SignAccess("user1", "ses1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "eposta") || strings.Contains(s, "@") || strings.Contains(lower, "email") {
		t.Fatal("erişim jetonunda kişisel veri izi")
	}
}
