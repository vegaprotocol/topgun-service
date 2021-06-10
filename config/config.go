// Package config contains structures used in retrieving app configuration
// from disk.
package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config describes the top level config file format.
type Config struct {
	// Listen specifies the IP address and port to listen on, e.g. 127.0.0.1:1234, 0.0.0.0:5678
	Listen string `yaml:"listen"`

	LogFormat     string `yaml:"logFormat"`
	LogLevel      string `yaml:"logLevel"`
	LogMethodName bool   `yaml:"logMethodName"`

	// Algorithm describes the sorting method for ordering participants
	Algorithm string `yaml:"algorithm"`

	// AlgorithmConfig describes any algorithm-specific config for filtering/sorting participants.
	AlgorithmConfig map[string]string `yaml:"algorithmConfig"`

	// Description describes the competition
	Description string `yaml:"description"`

	DefaultDisplay string `yaml:"defaultDisplay"`

	DefaultSort string `yaml:"defaultSort"`

	GracefulShutdownTimeout time.Duration `yaml:"gracefulShutdownTimeout"`

	Headers []string `yaml:"headers"`

	SocialURL *url.URL `yaml:"socialURL"`

	// VegaAsset ...
	VegaAsset string `yaml:"vegaAsset"`

	VegaGraphQLURL *url.URL `yaml:"vegaGraphQLURL"`

	VegaPoll time.Duration `yaml:"vegaPoll"`

	StartTime time.Time `yaml:"startTime"`
	EndTime   time.Time `yaml:"endTime"`
}

func CheckConfig(cfg Config) error {
	var e *multierror.Error

	if len(cfg.Listen) == 0 {
		e = multierror.Append(e, errors.New("missing: listen"))
	}
	if len(cfg.LogFormat) == 0 {
		e = multierror.Append(e, errors.New("missing: logFormat"))
	}
	if len(cfg.LogLevel) == 0 {
		e = multierror.Append(e, errors.New("missing: logLevel"))
	}
	if len(cfg.Algorithm) == 0 {
		e = multierror.Append(e, errors.New("missing: algorithm"))
	}
	if len(cfg.Description) == 0 {
		e = multierror.Append(e, errors.New("missing: description"))
	}
	if len(cfg.DefaultDisplay) == 0 {
		e = multierror.Append(e, errors.New("missing: defaultDisplay"))
	}
	if len(cfg.DefaultSort) == 0 {
		e = multierror.Append(e, errors.New("missing: defaultSort"))
	}
	if cfg.GracefulShutdownTimeout <= 0 {
		e = multierror.Append(e, errors.New("invalid: gracefulShutdownTimeout (should be greater than 0)"))
	}
	if len(cfg.Headers) == 0 {
		e = multierror.Append(e, errors.New("missing: headers"))
	}
	if cfg.SocialURL == nil || cfg.SocialURL.String() == "" {
		e = multierror.Append(e, errors.New("missing: socialURL"))
	}
	if len(cfg.VegaAsset) == 0 {
		e = multierror.Append(e, errors.New("missing: vegaAsset"))
	}
	if cfg.VegaGraphQLURL == nil || cfg.VegaGraphQLURL.String() == "" {
		e = multierror.Append(e, errors.New("missing: vegaGraphQLURL"))
	}
	if cfg.VegaPoll <= 0 {
		e = multierror.Append(e, errors.New("invalid: vegaPoll (should be greater than 0)"))
	}
	if cfg.StartTime.Before(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)) {
		e = multierror.Append(e, errors.New("missing/invalid: startTime"))
	}
	if cfg.EndTime.Before(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)) {
		e = multierror.Append(e, errors.New("missing/invalid: endTime"))
	}

	return e.ErrorOrNil()
}

func (c *Config) String() string {
	fmtStr := "Config{ " +
		"listen:%s, " +
		"algorithm:%s" +
		"algorithmConfig:%v" +
		"description:%s, " +
		"gracefulShutdownTimeout:%s, " +
		"headers:%v" +
		"socialURL:%s, " +
		"vegaAsset:%s, " +
		"vegaGraphQLURL:%s, " +
		"vegaPoll:%s" +
		"startTime:%s" +
		"endTime:%s" +
		"}"
	return fmt.Sprintf(
		fmtStr,
		c.Listen,
		c.Algorithm,
		c.AlgorithmConfig,
		c.Description,
		c.GracefulShutdownTimeout,
		c.Headers,
		c.SocialURL.String(),
		c.VegaAsset,
		c.VegaGraphQLURL.String(),
		c.VegaPoll.String(),
		c.StartTime,
		c.EndTime,
	)
}

func (c *Config) LogFields() log.Fields {
	return log.Fields{
		"listen":                  c.Listen,
		"logFormat":               c.LogFormat,
		"logLevel":                c.LogLevel,
		"logMethodName":           c.LogMethodName,
		"algorithm":               c.Algorithm,
		"algorithmConfig":         c.AlgorithmConfig,
		"description":             c.Description,
		"defaultDisplay":          c.DefaultDisplay,
		"defaultSort":             c.DefaultSort,
		"gracefulShutdownTimeout": c.GracefulShutdownTimeout,
		"headers":                 c.Headers,
		"socialURL":               c.SocialURL.String(),
		"vegaAsset":               c.VegaAsset,
		"vegaGraphQLURL":          c.VegaGraphQLURL.String(),
		"vegaPoll":                c.VegaPoll.String(),
		"startTime":               c.StartTime,
		"endTime":                 c.EndTime,
	}
}

func ConfigureLogging(cfg Config) error {
	// https://github.com/sirupsen/logrus#logging-method-name
	// This slows down logging (by a factor of 2).
	log.SetReportCaller(cfg.LogMethodName)

	switch cfg.LogFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "textcolour":
		log.SetFormatter(&log.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	case "textnocolour":
		log.SetFormatter(&log.TextFormatter{
			DisableColors:   true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	case "text":
		// with colour if TTY, without otherwise
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		return fmt.Errorf("invalid logFormat: %s", cfg.LogFormat)
	}

	loglevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "failed to set log level")
	}
	log.SetLevel(loglevel)
	return nil
}