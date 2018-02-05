package config

import (
	"fmt"
	"time"

	"github.com/urfave/cli"
)

const (
	DefaultTimeout = time.Minute * 15
	DefaultOutput  = "text"
)

func CLIFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   fmt.Sprintf("o, %s", FLAG_OUTPUT),
			EnvVar: FLAG_OUTPUT,
			Value:  DefaultOutput,
			Usage:  "output format [text,json]",
		},
		cli.DurationFlag{
			Name:   fmt.Sprintf("t, %s", FLAG_TIMEOUT),
			EnvVar: ENVVAR_TIMEOUT,
			Value:  DefaultTimeout,
			Usage:  "timeout [h,m,s,ms]",
		},
		cli.BoolFlag{
			Name:   fmt.Sprintf("d, %s", FLAG_DEBUG),
			EnvVar: ENVVAR_DEBUG,
			Usage:  "show debug output",
		},
		cli.StringFlag{
			Name:   FLAG_ENDPOINT,
			EnvVar: ENVVAR_ENDPOINT,
			Value:  "http://localhost:9090/",
			Usage:  "The endpoint of the Layer0 API",
		},
		cli.StringFlag{
			Name:   FLAG_TOKEN,
			EnvVar: ENVVAR_TOKEN,
			Usage:  "The auth token of the Layer0 API",
		},
		cli.BoolFlag{
			Name:   FLAG_SKIP_VERIFY_SSL,
			EnvVar: ENVVAR_SKIP_VERIFY_SSL,
			Usage:  "If set, will skip ssl verification",
		},
		cli.BoolFlag{
			Name:   FLAG_SKIP_VERIFY_VERSION,
			EnvVar: ENVVAR_SKIP_VERIFY_VERSION,
			Usage:  "If set, will skip version verification",
		},
	}
}

func ValidateCLIContext(c *cli.Context) error {
	requiredVars := []string{
		FLAG_TOKEN,
	}

	for _, name := range requiredVars {
		if !c.IsSet(name) {
			return fmt.Errorf("Required Variable '%s' is not set!", name)
		}
	}

	return nil
}