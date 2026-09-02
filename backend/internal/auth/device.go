package auth

import (
	"strings"
)

const deviceSep = "\x1e"

// DeviceFingerprint UA, dil ve cihaz kimliğinden SHA-256 üretir. Ham değer saklanmaz.
// X-Device-Id yoksa tarayıcı için User-Agent kimlik yerine geçer.
func DeviceFingerprint(deviceID, userAgent, acceptLanguage string) string {
	id := strings.TrimSpace(deviceID)
	ua := strings.TrimSpace(userAgent)
	lang := strings.TrimSpace(acceptLanguage)
	if id == "" {
		id = ua
	}
	return HashToken(ua + deviceSep + lang + deviceSep + id)
}

// DeviceLabel kullanıcı ajanından kısa bir ad üretir ("Chrome / macOS").
func DeviceLabel(userAgent string) string {
	ua := strings.TrimSpace(userAgent)
	if ua == "" {
		return "Bilinmeyen cihaz"
	}
	osName := deviceOS(ua)
	browser := deviceBrowser(ua)
	if browser == "" && osName == "" {
		return "Bilinmeyen cihaz"
	}
	if osName == "" {
		return browser
	}
	if browser == "" {
		return osName
	}
	return browser + " / " + osName
}

func deviceOS(ua string) string {
	switch {
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"):
		return "iOS"
	case strings.Contains(ua, "Mac OS X"), strings.Contains(ua, "Macintosh"):
		return "macOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	default:
		return ""
	}
}

func deviceBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "Edg/"):
		return "Edge"
	case strings.Contains(ua, "Electron"):
		return "Kontrata"
	case strings.Contains(ua, "Chrome/"):
		return "Chrome"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	case strings.Contains(ua, "Safari/"):
		return "Safari"
	default:
		return ""
	}
}
