package cmd

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"net"
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pdbogen/unifi2mqtt/types"
	"github.com/spf13/cobra"
)

func startMqtt(cmd *cobra.Command, ch <-chan types.Client,
	log zerolog.Logger) error {
	log = log.With().Str("module", "mqtt").Logger()

	host, _ := cmd.Flags().GetString("mqtt-host")
	prefix, _ := cmd.Flags().GetString("mqtt-prefix")
	port, _ := cmd.Flags().GetInt("mqtt-port")
	proto, _ := cmd.Flags().GetString("mqtt-proto")

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("checking MQTT config: %w", err)
	}
	conn.Close()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", proto, host, port))
	client := mqtt.NewClient(opts)

	log.Info().Msg("connecting to MQTT server...")
	token := client.Connect()
	token.WaitTimeout(time.Minute)
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt connect failed: %w", err)
	}
	if !client.IsConnected() {
		return errors.New("timeout waiting for mqtt to connect")
	}
	log.Info().Msg("connected to MQTT")

	go func(client mqtt.Client, prefix string, ch <-chan types.Client) {
		for device := range ch {
			payload := "not_home"
			if device.Present {
				payload = "home"
			}
			tok := client.Publish(filepath.Join(prefix, strings.ToLower(device.Name)), 1, true, payload)
			if err := tok.Error(); err != nil {
				log.Error().Err(err).Msg("error while publishing")
			}
		}
	}(client, prefix, ch)

	return nil
}
