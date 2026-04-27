package blocklist

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/user/go-reverse-proxy/internal/config"
)

// knownBotPaths are exact paths commonly scanned by bots/vulnerability scanners.
var knownBotPaths = map[string]bool{
	"/.favicon":               true,
	"/favicon.ico":            true,
	"/graphql/api":            true,
	"/graphql":                true,
	"/wordpress/admin.php":    true,
	"/wp-admin":               true,
	"/wp-admin/":              true,
	"/wp-login.php":           true,
	"/wp-config.php":          true,
	"/xmlrpc.php":             true,
	"/.env":                   true,
	"/.git":                   true,
	"/.git/config":            true,
	"/admin":                  true,
	"/admin/":                 true,
	"/administrator":          true,
	"/phpmyadmin":             true,
	"/phpmyadmin/":            true,
	"/pma":                    true,
	"/shell.php":              true,
	"/config.php":             true,
	"/setup.php":              true,
	"/install.php":            true,
	"/backup.php":             true,
	"/test.php":               true,
	"/info.php":               true,
	"/phpinfo.php":            true,
	"/login.php":              true,
	"/cgi-bin/":               true,
	"/boaform/admin/formLogin": true,
	"/.well-known/security.txt": true,
	"/sitemap.xml":            true,
	"/robots.txt":             true,
}

// knownBotPrefixes are path prefixes commonly targeted by bots.
var knownBotPrefixes = []string{
	"/wp-",
	"/wordpress/",
	"/.git/",
	"/vendor/",
	"/node_modules/",
	"/actuator/",
	"/solr/",
	"/jenkins/",
	"/jmx-console/",
	"/manager/",
	"/tomcat/",
	"/cgi-bin/",
	"/owa/",
	"/var/log/",
}

// knownBotSuffixes are file extensions typically probed by scanners.
var knownBotSuffixes = []string{
	".php",
	".asp",
	".aspx",
	".cgi",
	".jsp",
	".bak",
	".old",
	".sql",
	".tar",
	".tar.gz",
	".zip",
	".rar",
	".log",
	".cfg",
	".conf",
	".ini",
	".env",
	".DS_Store",
}

// suspicious user agents
var badUserAgents = []string{
	"curl",
	"wget",
	"python",
	"scanner",
	"sqlmap",
	"nikto",
	"masscan",
	"zgrab",
	"bot",
	"crawler",
	"spider",
}

var requestLog = make(map[string][]time.Time)
var requestLogMu sync.RWMutex

// isBot returns true if the request path matches a known bot/scanner pattern.
func isBot(path string) bool {
	lower := strings.ToLower(path)

	// Exact match
	if knownBotPaths[lower] {
		return true
	}

	// Prefix match
	for _, prefix := range knownBotPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	// Suffix match
	for _, suffix := range knownBotSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}

	return false
}

func isRateLimited(ip string) bool {
	cfg := config.GetConfig()
	now := time.Now()
	window := cfg.RateLimit.Window
	limit := cfg.RateLimit.Requests

	requestLogMu.Lock()
	timestamps := requestLog[ip]
	var valid []time.Time

	for _, t := range timestamps {
		if now.Sub(t) < window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= limit {
		requestLog[ip] = valid
		requestLogMu.Unlock()
		return true
	}

	valid = append(valid, now)
	requestLog[ip] = valid
	requestLogMu.Unlock()
	return false
}

func isBadUA(ua string) bool {
	ua = strings.ToLower(ua)
	for _, bad := range badUserAgents {
		if strings.Contains(ua, bad) {
			return true
		}
	}
	return false
}

// Middleware returns an http.Handler that blocks known bot/scanner paths.
// It wraps the given next handler and rejects suspicious requests with 403.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		ua := r.UserAgent()

		if isRateLimited(ip) {
			slog.Warn("blocked rate limited", "path", r.URL.Path, "ip", ip)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if isBadUA(ua) {
			slog.Warn("blocked bad user agent", "method", r.Method, "path", r.URL.Path, "ip", ip, "ua", ua)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if isBot(r.URL.Path) {
			slog.Warn("blocked bot/scanner", "method", r.Method, "path", r.URL.Path, "ip", ip)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CleanupRequestLog() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		now := time.Now()
		window := config.GetConfig().RateLimit.Window
		requestLogMu.Lock()
		for ip, timestamps := range requestLog {
			var valid []time.Time
			for _, t := range timestamps {
				if now.Sub(t) < window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(requestLog, ip)
			} else {
				requestLog[ip] = valid
			}
		}
		requestLogMu.Unlock()
	}
}
