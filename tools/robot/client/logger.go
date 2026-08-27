package client

import (
	"fmt"
	"github.com/yunjoy-tech/musae/logger"
)

func (c *Client) prefix() string {
	return fmt.Sprintf("[%s] [%s]", c.Account, c.GetState())
}

func (c *Client) Debug(args ...interface{}) {
	logger.Debug(c.prefix() + " " + fmt.Sprint(args...))
}

func (c *Client) Debugf(template string, args ...interface{}) {
	logger.Debugf(c.prefix()+" "+template, args...)
}

func (c *Client) Info(args ...interface{}) {
	logger.Info(c.prefix() + " " + fmt.Sprint(args...))
}

func (c *Client) Infof(template string, args ...interface{}) {
	logger.Infof(c.prefix()+" "+template, args...)
}

func (c *Client) Warn(args ...interface{}) {
	logger.Warn(c.prefix() + " " + fmt.Sprint(args...))
}

func (c *Client) Warnf(template string, args ...interface{}) {
	logger.Warnf(c.prefix()+" "+template, args...)
}

func (c *Client) Error(args ...interface{}) {
	logger.Error(c.prefix() + " " + fmt.Sprint(args...))
}

func (c *Client) Errorf(template string, args ...interface{}) {
	logger.Errorf(c.prefix()+" "+template, args...)
}
