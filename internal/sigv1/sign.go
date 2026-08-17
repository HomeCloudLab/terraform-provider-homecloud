// Package sigv1 implements HomeCloud SigV1 (HMAC-SHA256) for console API requests.
// Canonical string matches homecloud_auth.signature / homecloud_core.signing:
//
//	{METHOD}\n{path}\n{timestamp}\n{account_id}
package sigv1

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const (
	HeaderAccessKeyID = "X-Homecloud-Access-Key-Id"
	HeaderDate        = "X-Homecloud-Date"
	HeaderSignature   = "X-Homecloud-Signature"
	WhoamiAccountID   = "-"
)

func BuildStringToSign(method, path, timestamp, accountID string) string {
	return strings.ToUpper(method) + "\n" + path + "\n" + timestamp + "\n" + accountID
}

func ComputeSignature(secret, stringToSign string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil))
}

func FormatTimestamp(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}

func SignHeaders(accessKeyID, secret, method, path, accountID string, now time.Time) map[string]string {
	ts := FormatTimestamp(now)
	sig := ComputeSignature(secret, BuildStringToSign(method, path, ts, accountID))
	return map[string]string{
		HeaderAccessKeyID: accessKeyID,
		HeaderDate:        ts,
		HeaderSignature:   sig,
	}
}
