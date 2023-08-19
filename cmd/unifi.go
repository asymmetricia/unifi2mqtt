package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"github.com/pdbogen/unifi2mqtt/types"
	"github.com/spf13/cobra"
)

func startUnifi(cmd *cobra.Command, log zerolog.Logger) (<-chan types.Client, error) {
	log = log.With().Str("module", "unifi").Logger()

	verifyFlags(cmd, log)

	names, _ := cmd.Flags().GetStringSlice("include-name")
	matchers, err := types.NewMatchers(names)
	if err != nil {
		return nil, fmt.Errorf("parsing --include-name: %w", err)
	}

	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	verifyTls, _ := cmd.Flags().GetBool("verify-tls")
	deviceTimeout, _ := cmd.Flags().GetDuration("unifi-timeout")
	connectTimeout, _ := cmd.Flags().GetDuration("timeout")

	u, err := types.NewUnifi(username, password, host, port, verifyTls, deviceTimeout, connectTimeout, matchers, log)
	if err != nil {
		return nil, fmt.Errorf("starting unifi client: %w", err)
	}

	return u.Chan(), nil
}

func verifyFlags(cmd *cobra.Command, log zerolog.Logger) {
	for flag, env := range map[string]string{"username": "UNIFI2MQTT_USER", "password": "UNIFI2MQTT_PASS"} {
		flagV, _ := cmd.Flags().GetString(flag)
		if flagV == "" {
			flagV = os.Getenv(env)
		}
		if flagV == "" {
			log.Fatal().Msgf("flag --%s or environment variable %s is required", flag, env)
		}
		cmd.Flags().Set(flag, flagV)
	}
}
