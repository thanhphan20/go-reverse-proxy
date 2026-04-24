package blocklist

import (
	"log"
	"net/http"
	"strings"
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

// Middleware returns an http.Handler that blocks known bot/scanner paths.
// It wraps the given next handler and rejects suspicious requests with 403.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isBot(r.URL.Path) {
			log.Printf("[BLOCKED] bot/scanner request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
