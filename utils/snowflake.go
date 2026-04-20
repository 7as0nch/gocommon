// Package utils @author <7as0nch@gmail.com>
//
//	@date	2023/2/16
//	@note
package utils

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
)

var Node *snowflake.Node

const base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	// 设置起始时间 (Epoch)
	// 这里设置为 2025-01-01 00:00:00 UTC 的毫秒时间戳
	st := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	snowflake.Epoch = st.UnixMilli()
	// 获取本机IP
	var ip string
	if ips, err := net.LookupHost("localhost"); err == nil && len(ips) > 0 {
		ip = ips[0]
	} else {
		ip = "127.0.0.1"
	}
	var seed int
	var err error
	if ip == "" {
		seed = 1
	} else {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil || parsedIP.To4() == nil {
			seed = 1
		} else {
			arr := strings.Split(parsedIP.To4().String(), ".")
			lastOne := arr[3]
			seed, err = strconv.Atoi(lastOne)
			if err != nil {
				panic(err)
			}
		}
	}
	idGenerator, err := snowflake.NewNode(int64(seed))
	if err != nil {
		panic(err)
	}
	Node = idGenerator
}

func GetSFID() int64 {
	return Node.Generate().Int64()
}

// ToBase62 将十进制ID转为62进制字符串（字符集: 0-9a-zA-Z）
func ToBase62(id int64) string {
	if id == 0 {
		return "0"
	}
	negative := id < 0
	var num uint64
	if negative {
		// 兼容最小负数，避免直接取反溢出。
		num = uint64(-(id + 1))
		num++
	} else {
		num = uint64(id)
	}
	buf := make([]byte, 0, 12)
	for num > 0 {
		buf = append(buf, base62Alphabet[num%62])
		num /= 62
	}
	if negative {
		buf = append(buf, '-')
	}
	// 反转
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// GetSFIDBase62 生成雪花ID并转换为62进制短字符串, 分布式唯一，类似短链接
func GetSFIDBase62() string {
	return ToBase62(GetSFID())
}

// Base62ToSFID 将62进制短字符串转为雪花ID
func Base62ToSFID(base62 string) int64 {
	if base62 == "" {
		return 0
	}

	negative := false
	chars := []byte(base62)

	// 判断负号
	if chars[0] == '-' {
		negative = true
		chars = chars[1:]
	}
	var num uint64 = 0
	for _, c := range chars {
		var val int
		switch {
		case c >= '0' && c <= '9':
			val = int(c - '0')
		case c >= 'a' && c <= 'z':
			val = int(c-'a') + 10
		case c >= 'A' && c <= 'Z':
			val = int(c-'A') + 36
		default:
			// 非法字符忽略/抛异常
			return 0
		}
		num = num*62 + uint64(val)
	}

	if negative {
		// 还原负数编码规则
		if num == 0 {
			return 0
		}
		return -(int64(num) - 1) - 1
	}
	return int64(num)
}



