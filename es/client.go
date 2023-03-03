package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
)

type Client struct {
	client *elasticsearch.Client
}

func NewClient(address []string, user, password string) (*Client, error) {
	c, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: address,
		Username:  user,
		Password:  password,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Client{
		client: c,
	}, nil
}

type Config struct {
	Index         string
	Size          int
	RangeTimeName string
	Terms         []*Term
}

type Term struct {
	Key   string
	Value []string
}

// getMessageByRange 根据  gte~lte 获取相关 messages
// 一旦发现返回数据两超过 size，则利用 getMoreMessageByRange 获取更多数据
func (c *Client) GetMessageByRange(gte, lte int64, conf *Config) ([]json.RawMessage, error) {
	sb := newSearchBody(gte, lte, conf)
	fmt.Println(sb)
	res, err := c.client.Search(
		c.client.Search.WithContext(context.Background()),
		c.client.Search.WithIndex(conf.Index),
		c.client.Search.WithBody(bytes.NewBufferString(sb.String())),
		c.client.Search.WithTrackTotalHits(true),
		c.client.Search.WithSize(conf.Size),
		c.client.Search.WithSort(conf.RangeTimeName+":asc"),
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		err = json.NewDecoder(res.Body).Decode(&e)
		if err != nil {
			return nil, errors.WithMessage(err, "Error parsing the response body")
		}
		// Print the response status and error information.
		return nil, errors.Errorf("[%s] %s: %s",
			res.Status(),
			e["error"].(map[string]interface{})["type"],
			e["error"].(map[string]interface{})["reason"])
	}

	r := new(searchResponse)
	err = json.NewDecoder(res.Body).Decode(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if r.Hits.Total.Value <= conf.Size {
		return r.getSource(), nil
	}

	// r.Hits.Total > size
	return c.getMoreMessageByRange(gte, lte, r.Hits.Total.Value, conf)
}

// getMoreMessageByRange 通过将 gte~lte 拆分以减少每次查询的数量
func (c *Client) getMoreMessageByRange(gte, lte int64, total int, conf *Config) ([]json.RawMessage, error) {
	if lte-gte == 1 {
		// TODO
		conf.Size = total
	}
	k := total / conf.Size
	if k*conf.Size < total {
		k++
	}

	result := make([]json.RawMessage, 0, total)
	step := (lte - gte) / int64(k)
	if step < 1 {
		step = 1
	}
	for a := gte; a < lte; a += step {
		b := a + step
		if b > lte {
			b = lte
		}
		mid, err := c.GetMessageByRange(a, b, conf)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		result = append(result, mid...)
	}
	return result, nil
}
