package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// PhoneAlertNotifier handles phone alert notifications.
type PhoneAlertNotifier struct {
	Webhook string `json:"webhook"`
	AliyunField string `json:"aliyun_field"`
	TencentField string `json:"tencent_field"`
}

// Notify sends a phone alert if the message is critical.
func (p *PhoneAlertNotifier) Notify(msg AlertMessage) error {
	if msg.Level == "critical" {
		// Code to send phone alert
	}
	return nil
}