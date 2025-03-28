/*
*
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package sms

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/shangzongyu/mqant-modules/tools"
	"github.com/shangzongyu/mqant/conf"
	"github.com/shangzongyu/mqant/gate"
	"github.com/shangzongyu/mqant/module"
	"github.com/shangzongyu/mqant/module/base"
)

var Module = func() module.Module {
	user := new(SMS)
	return user
}

type SMS struct {
	basemodule.BaseModule
	RedisUrl  string
	TTL       int64
	SendCloud map[string]interface{}
	Ailyun    map[string]interface{}
}

func (sms *SMS) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "sms"
}

func (sms *SMS) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (sms *SMS) OnInit(app module.App, settings *conf.ModuleSettings) {
	sms.BaseModule.OnInit(sms, app, settings)
	sms.RedisUrl = sms.GetModuleSettings().Settings["RedisUrl"].(string)
	sms.TTL = int64(sms.GetModuleSettings().Settings["TTL"].(float64))
	if SendCloud, ok := sms.GetModuleSettings().Settings["SendCloud"]; ok {
		sms.SendCloud = SendCloud.(map[string]interface{})
	}
	if Ailyun, ok := sms.GetModuleSettings().Settings["Ailyun"]; ok {
		sms.Ailyun = Ailyun.(map[string]interface{})
	}
	sms.GetServer().RegisterGO("SendVerifiycode", sms.doSendVerifiycode) //演示后台模块间的rpc调用
	sms.GetServer().RegisterGO("GetCodeData", sms.getCodeData)           //演示后台模块间的rpc调用
}

func (sms *SMS) Run(closeSig chan bool) {
}

func (sms *SMS) OnDestroy() {
	//一定别忘了关闭RPC
	sms.GetServer().OnDestroy()
}

func (sms *SMS) aliyun(phone string, smsCode int64) string {
	param := map[string]string{
		"Action":        "SendSms",
		"Version":       "2017-05-25",
		"RegionId":      "cn-hangzhou",
		"PhoneNumbers":  phone,
		"SignName":      sms.Ailyun["SignName"].(string),
		"TemplateCode":  sms.Ailyun["TemplateCode"].(string),
		"TemplateParam": fmt.Sprintf("{\"smsCode\": \"%d\"}", smsCode),
		"OutId":         fmt.Sprintf("%d", time.Now().Unix()*1000),
	}
	AliyunPOPSignature("POST", sms.Ailyun["AccessKeyId"].(string), sms.Ailyun["AccessSecret"].(string), param)
	values := url.Values{}
	for k, v := range param {
		values[k] = []string{v}
	}
	//log.Error(values.Encode())
	req, err := http.PostForm("http://dysmsapi.aliyuncs.com", values)
	if err != nil {
		// handle error
		return err.Error()
	}

	//req, err := http.Get("http://dysmsapi.aliyuncs.com?"+values.Encode())
	//if err != nil {
	//	return err.Error()
	//}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	/**
	  # 部分成功
	  	{
	  	    "message":"部分成功",
	  	    "info":{
	  		    "successCount":1,
	  		    "failedCount":1,
	  		    "items":[{"phone":"1312222","vars":{},"message":"手机号格式错误"}],
	  		    "smsIds":["1458113381893_15_3_11_1ainnq$131112345678"]}
	  		    },
	  	    "result":true,
	  	    "statusCode":311
	  	}
	*/
	ret := map[string]interface{}{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return err.Error()
	}
	if result, ok := ret["Code"]; ok {
		if result.(string) != "OK" {
			if message, ok := ret["Message"]; ok {
				return message.(string)
			} else {
				return "验证码发送失败"
			}
		} else {
			return ""
		}
	} else {
		return string(body)
	}
}

func (sms *SMS) sendcloud(phone string, smsCode int64) string {
	param := map[string]string{
		"smsUser":    sms.SendCloud["SmsUser"].(string),
		"templateId": sms.SendCloud["TemplateId"].(string),
		"msgType":    "0",
		"phone":      phone,
		"vars":       fmt.Sprintf("{\"smsCode\": \"%d\"}", smsCode),
		"timestamp":  fmt.Sprintf("%d", time.Now().Unix()*1000),
	}
	SendCloudSignature(sms.SendCloud["SmsKey"].(string), param)
	values := url.Values{}
	for k, v := range param {
		values[k] = []string{v}
	}
	req, err := http.PostForm("http://www.sendcloud.net/smsapi/send", values)
	if err != nil {
		// handle error
		return err.Error()
	}

	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	/**
	  # 部分成功
	  	{
	  	    "message":"部分成功",
	  	    "info":{
	  		    "successCount":1,
	  		    "failedCount":1,
	  		    "items":[{"phone":"1312222","vars":{},"message":"手机号格式错误"}],
	  		    "smsIds":["1458113381893_15_3_11_1ainnq$131112345678"]}
	  		    },
	  	    "result":true,
	  	    "statusCode":311
	  	}
	*/
	ret := map[string]interface{}{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return err.Error()
	}
	if result, ok := ret["result"]; ok {
		if !result.(bool) {
			if message, ok := ret["message"]; ok {
				return message.(string)
			} else {
				return "验证码发送失败"
			}
		} else {
			return ""
		}
	} else {
		return string(body)
	}
}

/*
*
发送验证码
*/
func (sms *SMS) doSendVerifiycode(session gate.Session, phone string, purpose string, extra map[string]interface{}) (string, string) {
	conn := tools.GetRedisFactory().GetPool(sms.RedisUrl).Get()
	defer conn.Close()
	ttl, err := redis.Int64(conn.Do("TTL", fmt.Sprintf(MobileTTLFormat, phone)))
	if err != nil {
		return "", err.Error()
	}
	if ttl > 0 {
		return "", "操作过于频繁，请您稍后再试。"
	}

	smsCode := RandInt64(100000, 999999)
	if sms.Ailyun != nil {
		errstr := sms.aliyun(phone, smsCode)
		if errstr != "" {
			if sms.SendCloud != nil {
				errstr := sms.sendcloud(phone, smsCode)
				if errstr != "" {
					return "", errstr
				}
			} else {
				return "", errstr
			}
		}
	} else if sms.SendCloud != nil {
		errstr := sms.sendcloud(phone, smsCode)
		if errstr != "" {
			return "", errstr
		}
	} else {
		return "", "没有可用的短信通道。"
	}
	_, err = conn.Do("SET", fmt.Sprintf(MobileTTLFormat, phone), smsCode)
	if err != nil {
		return "", err.Error()
	}
	_, err = conn.Do("EXPIRE", fmt.Sprintf(MobileTTLFormat, phone), sms.TTL)
	if err != nil {
		return "", err.Error()
	}

	savedatas := map[string]interface{}{
		"purpose": purpose,
		"extra":   extra,
	}
	savedatasBytes, err := json.Marshal(savedatas)
	if err != nil {
		return "", err.Error()
	}
	_, err = conn.Do("SET", fmt.Sprintf(MobileSmsCodeFormat, phone, smsCode), savedatasBytes)
	if err != nil {
		return "", err.Error()
	}
	_, err = conn.Do("EXPIRE", fmt.Sprintf(MobileSmsCodeFormat, phone, smsCode), sms.TTL*5)
	if err != nil {
		return "", err.Error()
	}
	return "验证码发送成功", ""
}

/*
*
获取验证码参数
如果验证码已过期将返回失败
*/
func (sms *SMS) getCodeData(session gate.Session, phone string, smsCode int64, del bool) (map[string]interface{}, string) {
	conn := tools.GetRedisFactory().GetPool(sms.RedisUrl).Get()
	defer conn.Close()
	r, err := redis.Bytes(conn.Do("GET", fmt.Sprintf(MobileSmsCodeFormat, phone, smsCode)))
	if err != nil {
		return nil, err.Error()
	}
	if del {
		conn.Do("DEL", fmt.Sprintf(MobileSmsCodeFormat, phone, smsCode))
	}
	savedatas := map[string]interface{}{}
	err = json.Unmarshal(r, &savedatas)
	if err != nil {
		return nil, err.Error()
	}
	return savedatas, ""
}
