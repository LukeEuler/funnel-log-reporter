package flr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LukeEuler/funnel/common"
	"github.com/LukeEuler/funnel/event"
	"github.com/LukeEuler/funnel/model"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/LukeEuler/funnel-log-reporter/config"
	"github.com/LukeEuler/funnel-log-reporter/consumer"
	"github.com/LukeEuler/funnel-log-reporter/es"
	"github.com/LukeEuler/funnel-log-reporter/log"
)

const minDuration time.Duration = -1 << 63

type Processor struct {
	producer *es.Client
	consumer *consumer.Consumer

	lastLogs    int
	lastWhisper time.Time // show every day when no alers
	// 该字段的作用是为了防止重复报警
	lastEventTime int64

	lastMessages []json.RawMessage

	// do not alert when no new event in every group
	lastGroupEventsRecord map[string]int64
}

func NewProcessor() (*Processor, error) {
	conf := config.Conf
	p := &Processor{
		lastWhisper:  time.Now(),
		lastMessages: make([]json.RawMessage, 0),
	}

	var err error
	p.producer, err = es.NewClient(conf.Es.Address, conf.Es.Username, conf.Es.Password)
	if err != nil {
		return nil, err
	}

	p.consumer = new(consumer.Consumer)
	if conf.Ding.Enable {
		p.consumer.SetDingTalk(conf.Ding.URL, conf.Ding.Secret, conf.Ding.Mobiles)
	}
	if conf.Lark.Enable {
		p.consumer.SetLark(conf.Lark.URL, conf.Lark.Secret)
	}

	if conf.Hi {
		_ = p.consumer.Send("启动", "blue", "Kill time, or kiss time", false)
	}

	return p, nil
}

func (p *Processor) Loop(shutdown chan struct{}) {
	timer := time.NewTimer(minDuration)
	for {
		select {
		case <-shutdown:
			return
		case <-timer.C:
			p.work()
			timer.Reset(time.Duration(config.Conf.CheckInterval) * time.Second)
		}
	}
}

func (p *Processor) work() {
	conf := config.Conf
	endTime := time.Now().Unix()
	beginTime := endTime - conf.Duration

	// 增量更新数据, 减少es数据查询量
	esBeginTime := beginTime
	lastEndTime := lastValidEndTime(p.lastMessages, conf.Es.RangeTimeName)
	if esBeginTime < lastEndTime {
		esBeginTime = lastEndTime
	}

	newData, err := p.producer.GetMessageByRange(esBeginTime*1000, endTime*1000, conf.ToEsConfig())
	if err != nil {
		log.Entry.WithError(err).Error(err)
		return
	}

	p.lastMessages, err = lastValidMsg(p.lastMessages, conf.Es.RangeTimeName, beginTime)
	if err != nil {
		log.Entry.WithError(err).Error(err)
		p.lastMessages = newData
	} else {
		p.lastMessages = append(p.lastMessages, newData...)
	}

	// 格式化数据
	message := make([]model.EventData, 0, len(p.lastMessages))
	for _, item := range p.lastMessages {
		message = append(message,
			common.NewJSONData(item).
				SetTimeKeys(conf.TimeKey...))
	}

	log.Entry.Warnf("get %d message", len(message))

	events, err := event.Draw(message, conf.GetRules())
	if err != nil {
		log.Entry.WithError(err).Error(err)
		return
	}

	length := len(events)
	if length == 0 {
		if p.lastLogs == 0 {
			if time.Since(p.lastWhisper) > 24*time.Hour {
				err = p.consumer.Send("heartbeat", "wathet", "а зори здесь тихие. 🤫", false)
				if err != nil {
					log.Entry.WithError(err).Error(err)
				}
				p.lastWhisper = time.Now()
			}
			return
		}
		p.lastLogs = 0
		p.lastWhisper = time.Now()

		err = p.consumer.Send(
			"恢复",
			"green",
			fmt.Sprintf("tips: %s", conf.GetBaseQueryTimeInfo()),
			false)
		if err != nil {
			log.Entry.WithError(err).Error(err)
		}
		return
	}

	log.Entry.Warnf("%d needs report", len(events))
	duration := time.Duration(conf.Duration) * time.Second
	interval := time.Duration(conf.CheckInterval) * time.Second
	title := fmt.Sprintf("错误: %d/%d(有效/总数) in %s. interval %s",
		len(events), len(message), duration.String(), interval.String())

	change := false
	for _, item := range events {
		if item.GetTime() > p.lastEventTime {
			p.lastEventTime = item.GetTime()
			change = true
		}
	}

	if !change {
		return
	}

	content, ok := p.groupLogs(events, conf.GroupKeys, conf.ShowKeys)
	if !ok {
		return
	}
	err = p.consumer.Send(title, "red", content, true)
	if err != nil {
		log.Entry.WithError(err).Error(err)
		return
	}

	p.lastLogs = length
}

const (
	sep = "__"
)

func (p *Processor) groupLogs(events []model.Event, groupKeys [][]string, showKeys []string) (string, bool) {
	groupEventsRecord, collection := handleEvents(events, groupKeys)

	ok := p.checkAndReplaceGroupLogsRecord(groupEventsRecord)
	if !ok {
		return "", false
	}

	type tempRecord struct {
		groupTag string
		lastTime int64
	}
	sortList := make([]tempRecord, 0, len(groupEventsRecord))
	for groupTag, lastTime := range groupEventsRecord {
		sortList = append(sortList, tempRecord{groupTag, lastTime})
	}
	sort.SliceStable(sortList, func(i, j int) bool {
		return sortList[i].lastTime < sortList[j].lastTime
	})

	buffer := bytes.NewBufferString("")

	for idx, item := range sortList {
		groupTag := item.groupTag
		list := collection[groupTag]
		length := len(list)
		tagList := strings.Split(groupTag, sep)
		if idx > 0 {
			buffer.WriteString("\n")
		}
		buffer.WriteString(fmt.Sprintf("%v errors %d\n", tagList, length))
		for _, key := range showKeys {
			value, ok := list[length-1].GetValueString(key)
			if ok {
				buffer.WriteString(fmt.Sprintf("%s: %s\n", key, strings.TrimSpace(value)))
			} else {
				buffer.WriteString(fmt.Sprintf("%s -\n", key))
			}
		}
	}
	return buffer.String(), true
}

func handleEvents(events []model.Event, groupKeys [][]string) (map[string]int64, map[string][]model.Event) {
	collection := make(map[string][]model.Event)
	groupEventsRecord := make(map[string]int64)

	for _, item := range events {
		if !item.Valid() {
			continue
		}
		groupTags := make([]string, 0, len(groupKeys))
		for _, keys := range groupKeys {
			defaultKeyName := keys[0]
			var value string
			for _, key := range keys {
				var ok bool
				value, ok = item.GetValueString(key)
				if ok {
					break
				}
				value = "unknow " + defaultKeyName
			}
			groupTags = append(groupTags, value)
		}
		groupTag := strings.Join(groupTags, sep)

		_, ok := collection[groupTag]
		if !ok {
			collection[groupTag] = make([]model.Event, 0, 10)
		}
		_, ok = groupEventsRecord[groupTag]
		if !ok {
			groupEventsRecord[groupTag] = item.GetTime()
		}

		collection[groupTag] = append(collection[groupTag], item)
		if groupEventsRecord[groupTag] < item.GetTime() {
			groupEventsRecord[groupTag] = item.GetTime()
		}
	}
	return groupEventsRecord, collection
}

func (p *Processor) checkAndReplaceGroupLogsRecord(groupEventsRecord map[string]int64) bool {
	oldGroupEventsRecord := p.lastGroupEventsRecord
	p.lastGroupEventsRecord = groupEventsRecord

	if len(oldGroupEventsRecord) < len(groupEventsRecord) {
		return true
	}

	for key, recordTime := range groupEventsRecord {
		oldRecordTime, ok := oldGroupEventsRecord[key]
		if !ok {
			return true
		}
		if oldRecordTime != recordTime {
			return true
		}
	}
	return false
}

func lastValidEndTime(lastMessages []json.RawMessage, timeKey string) int64 {
	if len(lastMessages) == 0 {
		return 0
	}
	message := lastMessages[len(lastMessages)-1]
	value := gjson.GetBytes(message, timeKey)
	var t time.Time
	t, err := time.Parse(time.RFC3339, value.String())
	if err != nil {
		return 0
	}
	return t.Unix()
}

func lastValidMsg(lastMessages []json.RawMessage, timeKey string, begin int64) ([]json.RawMessage, error) {
	if len(lastMessages) == 0 {
		return nil, errors.New("last query data is empty")
	}

	idx := -1
	for i, message := range lastMessages {
		value := gjson.GetBytes(message, timeKey)
		var t time.Time
		t, err := time.Parse(time.RFC3339, value.String())
		if err != nil {
			return nil, errors.Errorf("can not parse %s as time", value)
		}
		if t.Unix() >= begin {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []json.RawMessage{}, nil
	}
	return lastMessages[idx:], nil
}
