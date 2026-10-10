package main

import "github.com/locallhosts/Wraith/backend-go/auth"

func rateLimitIdentityKey(id auth.Identity) string {
	if id.KeyID != "" {
		return "api-key:" + id.KeyID
	}
	if id.Label != "" {
		return "label:" + id.Label
	}
	return ""
}
