package consumer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type ding struct {
	url     string
	secret  string
	mobiles []string // at someones
}

func newDing(url, secret string, mobiles []string) *ding {
	return &ding{
		url:     url,
		secret:  secret,
		mobiles: mobiles,
	}
}

func (d *ding) Send(title, body string, notify bool) error {
	content := title + "\n\n" + body

	timestamp := time.Now().Unix() * 1000
	stringTimestamp := strconv.FormatInt(timestamp, 10)
	stringToSign := stringTimestamp + "\n" + d.secret
	signData := ghmac([]byte(d.secret), []byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(signData)

	temp := &requestBody{
		MsgType: "text",
		Text: requestText{
			Content: content,
		},
	}
	if notify {
		temp.At.AtMobiles = d.mobiles
	}

	bs, _ := json.Marshal(temp)

	req, err := http.NewRequest(http.MethodPost, d.url, bytes.NewBuffer(bs))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Add("Content-Type", "application/json")

	values := req.URL.Query()
	values.Set("timestamp", stringTimestamp)
	values.Set("sign", sign)
	req.URL.RawQuery = values.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	fmt.Println(string(result))
	return nil
}

type requestBody struct {
	MsgType string      `json:"msgtype"`
	Text    requestText `json:"text"`
	At      struct {
		AtMobiles []string `json:"atMobiles,omitempty"`
		IsAtAll   bool     `json:"isAtAll"`
	} `json:"at,omitempty"`
}

type requestText struct {
	Content string `json:"content"`
}

func ghmac(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}
