package config

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/LukeEuler/funnel/common"
	"github.com/LukeEuler/funnel/model"

	"github.com/LukeEuler/funnel-log-reporter/es"
	"github.com/LukeEuler/funnel-log-reporter/log"
)

var Conf *Config

func New(configPath string) {
	Conf = new(Config)
	_, err := toml.DecodeFile(configPath, Conf)
	if err != nil {
		log.Entry.Fatal(err)
	}
}

type Config struct {
	CheckInterval int64      `toml:"check_interval_s"` // 最小 10s
	Duration      int64      `toml:"duration_s"`
	GroupKeys     [][]string `toml:"group_keys"`
	ShowKeys      []string   `toml:"show_keys"`
	TimeKey       []string   `toml:"time_key"`
	Hi            bool       `toml:"hi"`
	Custom        struct {
		HiTitle               string `toml:"hi_title"`
		HiColor               string `toml:"hi_color"`
		HiContent             string `toml:"hi_content"`
		HeartbeatTitle        string `toml:"heartbeat_title"`
		HeartbeatTitleColor   string `toml:"heartbeat_title_color"`
		HeartbeatTitleContent string `toml:"heartbeat_title_content"`
		AlertColor            string `toml:"alert_color"`
		RecoverTitle          string `toml:"recover_title"`
		RecoverColor          string `toml:"recover_color"`
	} `toml:"custom"`
	Es struct {
		Address       []string `toml:"address"`
		Username      string   `toml:"username"`
		Password      string   `toml:"password"`
		Index         string   `toml:"index"`
		Size          int      `toml:"size"`
		RangeTimeName string   `toml:"range_time_name"`
		Term          []struct {
			Key    string   `toml:"key"`
			Values []string `toml:"values"`
		} `toml:"term"`
	} `toml:"es"`
	Ding struct {
		Enable  bool     `toml:"enable"`
		URL     string   `toml:"url"`
		Secret  string   `toml:"secret"`
		Mobiles []string `toml:"mobiles"`
	} `toml:"ding"`
	Lark struct {
		Enable bool   `toml:"enable"`
		URL    string `toml:"url"`
		Secret string `toml:"secret"`
	} `toml:"lark"`

	Rules map[string]*rule `toml:"rules"`
}

type rule struct {
	Name    string `toml:"name"`
	Content string `toml:"content"`
	Level   int    `toml:"level"`
	Mutex   bool   `toml:"mutex"`

	Start int64 `toml:"start"`
	End   int64 `toml:"end"`

	Duration int64 `toml:"duration"`
	Times    int   `toml:"times"`
}

func (r *rule) toEventRuleInfo(id string) *common.EventRuleInfo {
	return &common.EventRuleInfo{
		RuleInfo: &common.RuleInfo{
			ID:      id,
			Name:    r.Name,
			Content: r.Content,
		},
		Level:    r.Level,
		Mutex:    r.Mutex,
		Start:    r.Start,
		End:      r.End,
		Duration: r.Duration,
		Times:    r.Times,
	}
}

func (c *Config) ToEsConfig() *es.Config {
	esConf := &es.Config{
		Index:         c.Es.Index,
		Size:          c.Es.Size,
		RangeTimeName: c.Es.RangeTimeName,
		Terms:         make([]*es.Term, 0, len(c.Es.Term)),
	}

	for _, item := range c.Es.Term {
		esConf.Terms = append(esConf.Terms, &es.Term{
			Key:   item.Key,
			Value: item.Values,
		})
	}
	return esConf
}

func (c *Config) GetRules() []model.EventRule {
	result := make([]model.EventRule, 0, len(c.Rules))
	for id, item := range c.Rules {
		temp := item.toEventRuleInfo(id)
		result = append(result, temp)
	}
	return result
}

func (c *Config) GetBaseQueryTimeInfo() string {
	duration := time.Duration(c.Duration) * time.Second
	interval := time.Duration(c.CheckInterval) * time.Second
	return fmt.Sprintf("每次日志查询的时间范围: %s\n每次查询的时间间隔: %s\n",
		duration.String(),
		interval.String())
}
