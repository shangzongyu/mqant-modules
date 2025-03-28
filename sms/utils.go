// Copyright 2014 liangdas Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sms

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/shangzongyu/mqant/log"
	"github.com/shangzongyu/mqant/utils/uuid"
)

func RandInt64(min, max int64) int64 {
	if min >= max {
		return max
	}
	return rand.Int63n(max-min) + min
}
func SendCloudSignature(smsKey string, param map[string]string) string {
	delete(param, "signature")
	sortedKeys := make([]string, 0)
	for k, _ := range param {
		sortedKeys = append(sortedKeys, k)
	}

	// sort 'string' key in increasing order
	sort.Strings(sortedKeys)

	paramStrs := make([]string, 0)
	for _, k := range sortedKeys {
		paramStrs = append(paramStrs, k+"="+param[k])
	}
	paramStr := strings.Join(paramStrs, "&")
	signStr := smsKey + "&" + paramStr + "&" + smsKey
	h := md5.New()
	h.Write([]byte(signStr)) // 需要加密的字符串
	signature := h.Sum(nil)
	sign := hex.EncodeToString(signature)
	param["signature"] = sign
	return sign
}

func specialUrlEncode(value string) string {
	value = url.QueryEscape(value)
	value = strings.Replace(value, "+", "%20", -1)
	value = strings.Replace(value, "*", "%2A", -1)
	value = strings.Replace(value, "%7E", "~", -1)
	return value
}

func AliyunPOPSignature(HTTPMethod, accessKeyId string, accessSecret string, param map[string]string) string {
	delete(param, "Signature")
	local, err3 := time.LoadLocation("GMT")
	if err3 != nil {
		log.Error(err3.Error())
	}
	// 1. 系统参数
	param["SignatureMethod"] = "HMAC-SHA1"
	param["SignatureNonce"] = uuid.Rand().Hex()
	param["AccessKeyId"] = accessKeyId
	param["SignatureVersion"] = "1.0"
	param["Timestamp"] = time.Now().In(local).Format("2006-01-02T15:04:05Z") //"yyyy-MM-dd'T'HH:mm:ss'Z' 这里一定要设置GMT时区"
	param["Format"] = "JSON"

	//log.Error(time.Now().Format("2006-01-02T15:04:05Z"))
	//log.Error(time.Now().In(local).Format("2006-01-02T15:04:05Z"))
	sortedKeys := make([]string, 0)
	for k, _ := range param {
		sortedKeys = append(sortedKeys, k)
	}

	// sort 'string' key in increasing order
	sort.Strings(sortedKeys)

	paramStrs := make([]string, 0)
	for _, k := range sortedKeys {
		paramStrs = append(paramStrs, specialUrlEncode(k)+"="+specialUrlEncode(param[k]))
	}
	sortedQueryString := strings.Join(paramStrs, "&")
	//log.Error(sortedQueryString)
	//HTTPMethod + “&” + specialUrlEncode(“/”) + ”&” + specialUrlEncode(sortedQueryString)
	stringToSign := HTTPMethod + "&" + specialUrlEncode("/") + "&" + specialUrlEncode(sortedQueryString)
	//log.Error(stringToSign)
	hash := hmac.New(sha1.New, []byte(accessSecret+"&"))
	hash.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	//signature= specialUrlEncode(signature)
	//log.Error(signature)
	param["Signature"] = signature

	//for k, v := range param {
	//	param[k] = specialUrlEncode(v)
	//}

	//return	signature
	return signature
}
