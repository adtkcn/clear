package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtSecret = []byte("asdasjkasioasndlkahsdhalksd")

// Claims 生成token结构体
type Claims struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	jwt.StandardClaims
}

// GenerateToken 创建token
func GenerateToken(ID, Name string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(3 * time.Hour)

	claims := Claims{
		ID,
		Name,
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "gin-blog",
		},
	}
	// SigningMethodHS256、SigningMethodHS384、SigningMethodHS512
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtSecret)

	return token, err
}

// ParseToken 解析token
func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}

	return nil, err
}

// CheckExp 检查token是否有效
func CheckExp(token string) (*Claims, bool) {
	claims, err := ParseToken(token)
	if err != nil {
		fmt.Println(err)
		return claims, false
	}
	nowTime := time.Now().Unix()
	if claims.ExpiresAt < nowTime {
		fmt.Println("登录过期")
		return claims, false
	}
	fmt.Println("登录没过期")
	return claims, true
}

func FileHash(data []byte) string {
	m := md5.New()
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

// 查找切片arr中是否含有val
func Find(arr []string, val string) bool {
	var flag bool = false
	for _, value := range arr {
		fmt.Println(value, val, value == val)
		if value == val {
			flag = true
			break
		}
	}
	fmt.Println("fileExt", flag)
	return flag
}

// 打开浏览器
func OpenBrowser(urlAddr string) {
	fmt.Println("请打开地址：", urlAddr)
	if runtime.GOOS == "windows" {
		exec.Command(`cmd`, `/c`, `start`, urlAddr).Start()
	} else if runtime.GOOS == "linux" {
		exec.Command(`xdg-open`, urlAddr).Start()
	} else if runtime.GOOS == "darwin" {
		exec.Command(`open`, urlAddr).Start()
	}
}
