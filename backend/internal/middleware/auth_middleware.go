package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"backend/internal/domain"
)

const ContextUserKey = "currentUser"

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenString := ""
		if header != "" && strings.HasPrefix(header, "Bearer ") {
			tokenString = strings.TrimPrefix(header, "Bearer ")
		} else if q := c.Query("token"); q != "" {
			// EventSource cannot send custom headers; allow ?token= fallback.
			tokenString = q
		}
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid authorization header"})
			return
		}
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		sub, _ := claims["sub"].(string)
		name, _ := claims["name"].(string)
		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token subject"})
			return
		}

		user := &domain.User{
			ID:    sub,
			Name:  name,
			Email: email,
			Role:  domain.UserRole(role),
		}
		c.Set(ContextUserKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *domain.User {
	user, exists := c.Get(ContextUserKey)
	if !exists {
		return nil
	}
	u, ok := user.(*domain.User)
	if !ok {
		return nil
	}
	return u
}

func RequireRoles(roles ...domain.UserRole) gin.HandlerFunc {
	allowed := make(map[domain.UserRole]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}

	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if !allowed[user.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient role"})
			return
		}
		c.Next()
	}
}
