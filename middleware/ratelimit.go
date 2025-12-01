package middleware


import (
	"log" 
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	MaxRequestsPerIP  = 20
	WindowDuration    = 60 * time.Second
)

type ipAccess struct {
	Count int
	WindowEnd time.Time
}

var rateLimitStore = make(map[string]ipAccess)
var limitMutex sync.Mutex

func CheckAndIncrementAccess(ip string) bool {
	limitMutex.Lock()
	defer limitMutex.Unlock()

	now := time.Now()
	access, exists := rateLimitStore[ip]

	if !exists || now.After(access.WindowEnd) {
		rateLimitStore[ip] = ipAccess{
			Count: 1,
			WindowEnd: now.Add(WindowDuration),
		}
		return true 
	}

	if access.Count < MaxRequestsPerIP {
		access.Count++
		rateLimitStore[ip] = access
		return true
	}

	return false 
}

func GetClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	ip := strings.Split(r.RemoteAddr, ":")[0]
	return ip
}

func RateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := GetClientIP(c.Request)
		
		if !CheckAndIncrementAccess(clientIP) {
			c.Header("Retry-After", "60")
			log.Printf("GIN RATE LIMIT: IP %s exceeded limit of %d requests per %s.", clientIP, MaxRequestsPerIP, WindowDuration)
			c.String(http.StatusTooManyRequests, "Rate limit exceeded. Try again in 60 seconds.")
			c.Abort() 
			return
		}

		c.Next() 
	}
}
