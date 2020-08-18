package cmd

import (
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	Use:   "unifi2mqtt",
	Short: "a tool to poll the UniFi controller API and populate clients to mqtt",
	Long: "unifi2mqtt is a small gateway daemon in the spirit of " +
		"zwave2mqtt, which polls the UniFi Controller API and feeds " +
		"information about connected devices into MQTT, for the intended " +
		"purpose of powering HomeAssistant's MQTT Device Tracker.",
	Run: rootRun,
}

func init() {
	Root.Flags().String("host", "localhost", "hostname of the UniFi "+
		"controller")
	Root.Flags().Int("port", 8443, "UniFi controller port")
	Root.Flags().Bool("verify-tls", false, "if true, verify the TLS "+
		"certificate of the UniFi controller")
	Root.Flags().String("username", "", "login user on UniFi controller "+
		"(env: UNIFI2MQTT_USER)")
	Root.Flags().String("password", "", "user password on UniFi controller "+
		"(env: UNIFI2MQTT_PASS)")
	Root.Flags().StringSlice("include-name", nil, "client names to include; "+
		"if no --include flags are set, all clients will be reported. If a "+
		"name is surrounded with `/`, it's interpreted as a regular "+
		"expression. Flag may be specified multiple times, or just once "+
		"with `,`-separated options.")
	Root.Flags().Duration("unifi-timeout", time.Minute, "how long does a "+
		"device need to be not 'seen' before it's repoted as not_home. Note "+
		"that since unifi2mqtt _polls_ the unifi API once a minute, there's "+
		"no point in setting this lower than that. Unifi itself will stop "+
		"reporting a device after about five minutes.")

	Root.Flags().String("mqtt-host", "localhost", "hostname of the MQTT server to publish to")
	Root.Flags().String("mqtt-prefix", "unifi", "a prefix for the mqtt "+
		"space to publish, messages are published to "+
		"{mqtt-prefix}/{device-name}")
	Root.Flags().Int("mqtt-port", 1883, "port used to connect to MQTT")
	Root.Flags().String("mqtt-proto", "tcp", "protocol used to connect to MQTT; options are `tcp`, `ssl`, `ws`")
}

func rootRun(cmd *cobra.Command, _ []string) {
	var log zerolog.Logger

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		log = zerolog.New(zerolog.NewConsoleWriter())
	} else {
		log = zerolog.New(os.Stdout)
	}
	log = log.With().Timestamp().Logger()

	log.Info().Str("version", cmd.Root().Version).Msg("unifi2mqtt starting...")

	ch, err := startUnifi(cmd, log)
	if err != nil {
		log.Fatal().Err(err).Msg("could not connect to unifi controller")
	}

	if err := startMqtt(cmd, ch, log); err != nil {
		log.Fatal().Err(err).Msg("could not start mqtt client")
	}

	// block forever
	select {}
}
