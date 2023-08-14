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

type lark struct {
	url    string
	secret string
}

func newLark(url, secret string) *lark {
	return &lark{
		url:    url,
		secret: secret,
	}
}

func (l *lark) Send(title, color, content string) error {
	// fmt.Println(title)
	// fmt.Println(content)
	// return nil
	temp := &cardBody{
		MsgType: "interactive",
	}
	temp.Card.Config.WideScreenMode = true

	temp.Card.Header.Title.Tag = "plain_text"
	temp.Card.Header.Title.Content = title
	temp.Card.Header.Template = color

	item := cardElement{Tag: "div"}
	item.Text.Tag = "plain_text"
	item.Text.Content = content
	temp.Card.Elements = []cardElement{item}

	if len(l.secret) > 0 {
		timestamp := time.Now().Unix()
		temp.Timestamp = strconv.Itoa(int(timestamp))
		stringToSign := temp.Timestamp + "\n" + l.secret

		var data []byte
		h := hmac.New(sha256.New, []byte(stringToSign))
		_, err := h.Write(data)
		if err != nil {
			return errors.WithStack(err)
		}

		signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
		temp.Sign = signature
	}

	bs, _ := json.Marshal(temp)

	req, err := http.NewRequest(http.MethodPost, l.url, bytes.NewBuffer(bs))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	fmt.Println(string(result))
	return nil
}

type cardBody struct {
	Timestamp string `json:"timestamp,omitempty"`
	Sign      string `json:"sign,omitempty"`
	MsgType   string `json:"msg_type"`
	Card      struct {
		Config struct {
			WideScreenMode bool `json:"wide_screen_mode"`
		} `json:"config"`
		Header struct {
			Title struct {
				Tag     string `json:"tag"`
				Content string `json:"content"`
			} `json:"title"`
			// https://open.larksuite.com/document/ukTMukTMukTM/ukTNwUjL5UDM14SO1ATN
			Template string `json:"template"`
		}
		Elements []cardElement `json:"elements"`
	} `json:"card"`
}

type cardElement struct {
	Tag  string `json:"tag"`
	Text struct {
		Tag     string `json:"tag"`
		Content string `json:"content"`
	} `json:"text"`
}
