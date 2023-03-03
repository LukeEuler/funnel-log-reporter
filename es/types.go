package es

import "encoding/json"

// request

type searchBody struct {
	Query struct {
		Bool struct {
			Must []interface{} `json:"must"`
		} `json:"bool"`
	} `json:"query"`
}

func (b *searchBody) String() string {
	bs, _ := json.Marshal(b)
	return string(bs)
}

type searchRange struct {
	Range map[string]*timeRange `json:"range"`
}

type timeRange struct {
	Gte int64 `json:"gte"`
	Lte int64 `json:"lte"`
}

type searchTerms struct {
	Terms map[string][]string `json:"terms"`
}

func newSearchBody(gte, lte int64, conf *Config) *searchBody {
	sb := new(searchBody)
	sb.Query.Bool.Must = make([]interface{}, 0, 1+len(conf.Terms))
	sr := &searchRange{
		Range: make(map[string]*timeRange, 1),
	}
	sr.Range[conf.RangeTimeName] = &timeRange{
		Gte: gte,
		Lte: lte,
	}
	sb.Query.Bool.Must = append(sb.Query.Bool.Must, sr)

	for _, item := range conf.Terms {
		st := &searchTerms{
			Terms: map[string][]string{item.Key: item.Value},
		}
		sb.Query.Bool.Must = append(sb.Query.Bool.Must, st)
	}
	return sb
}

// response

type searchResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore json.Number `json:"max_score"`
		Hits     []*struct {
			Index  string          `json:"_index"`
			Type   string          `json:"_type"`
			ID     string          `json:"_id"`
			Score  json.Number     `json:"_score"`
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (r *searchResponse) getSource() []json.RawMessage {
	result := make([]json.RawMessage, 0, len(r.Hits.Hits))
	for i := range r.Hits.Hits {
		result = append(result, r.Hits.Hits[i].Source)
	}
	return result
}
