package mqtt

import (
    "time"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
    client mqtt.Client
}

type Config struct {
    BrokerURL string
    Username  string
    Password  string
    ClientID  string
}

func New(config Config) (*Client, error) {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(config.BrokerURL)
    opts.SetClientID(config.ClientID)
    opts.SetConnectTimeout(10 * time.Second)
    if config.Username != "" {
        opts.SetUsername(config.Username)
        opts.SetPassword(config.Password)
    }

    client := mqtt.NewClient(opts)
    token := client.Connect()
    if token.Wait() && token.Error() != nil {
        return nil, token.Error()
    }

    return &Client{client: client}, nil
}

func (c *Client) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) error {
    token := c.client.Subscribe(topic, qos, handler)
    if token.Wait() && token.Error() != nil {
        return token.Error()
    }
    return nil
}

func (c *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
    token := c.client.Publish(topic, qos, retained, payload)
    if token.Wait() && token.Error() != nil {
        return token.Error()
    }
    return nil
}

func (c *Client) Close() {
    c.client.Disconnect(250)
}
