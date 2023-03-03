package consumer

type Consumer struct {
	larkEnable, dingEnable bool

	larkClient *lark
	dingClient *ding
}

func (c *Consumer) SetLark(url, secret string) {
	c.larkClient = newLark(url, secret)
	c.larkEnable = true
}

func (c *Consumer) SetDingTalk(url, secret string, mobiles []string) {
	c.dingClient = newDing(url, secret, mobiles)
	c.dingEnable = true
}

func (c *Consumer) Send(title, color, content string, notify bool) error {
	var dingErr, larkErr error
	if c.larkEnable {
		dingErr = c.larkClient.Send(title, color, content)
	}
	if c.dingEnable {
		larkErr = c.dingClient.Send(title, content, notify)
	}
	if dingErr != nil {
		return dingErr
	}
	return larkErr
}
