package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
)

func (s *ChatGPTRegisterService) chatGPTWebAccountStats(ctx context.Context) (quota int, available int) {
	if s == nil || s.accounts == nil {
		return 0, 0
	}
	accounts, err := s.accounts.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return 0, 0
	}
	return chatGPTRegisterComputeAccountStats(accounts)
}

func chatGPTRegisterComputeAccountStats(accounts []Account) (quota int, available int) {
	for i := range accounts {
		acc := &accounts[i]
		if acc == nil || !IsChatGPTWebPoolAccount(acc) || !chatGPTRegisterAccountAvailable(acc) {
			continue
		}
		available++
		quota += chatGPTRegisterAccountQuota(acc)
	}
	return quota, available
}

func chatGPTRegisterAccountAvailable(acc *Account) bool {
	if acc == nil {
		return false
	}
	switch acc.Status {
	case StatusActive, "正常", "":
		return true
	default:
		return false
	}
}

func chatGPTRegisterAccountQuota(acc *Account) int {
	if acc == nil {
		return 0
	}
	for _, key := range []string{"quota", "current_quota", "available_quota", "chatgpt_quota"} {
		if v, ok := acc.Extra[key]; ok {
			if n := parseRegisterInt(v); n > 0 {
				return n
			}
		}
		if v, ok := acc.Credentials[key]; ok {
			if n := parseRegisterInt(v); n > 0 {
				return n
			}
		}
	}
	limit, used := acc.GetQuotaLimit(), acc.GetQuotaUsed()
	if limit > used {
		return int(math.Round(limit - used))
	}
	return 0
}

func parseRegisterInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(math.Round(x))
	case float32:
		return int(math.Round(float64(x)))
	case jsonNumberLike:
		n, _ := strconv.Atoi(x.String())
		return n
	case string:
		n, _ := strconv.Atoi(x)
		return n
	default:
		n, _ := strconv.Atoi(fmt.Sprint(v))
		return n
	}
}

type jsonNumberLike interface{ String() string }
