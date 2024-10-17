package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/AnkushinDaniil/interferometer/app"
	"github.com/AnkushinDaniil/interferometer/entity/format"
	"github.com/AnkushinDaniil/interferometer/entity/mode"
	"github.com/AnkushinDaniil/interferometer/entity/parameters"
)

const (
	modeF         = "mode"
	formatF       = "format"
	sourceF       = "source"
	outputF       = "output"
	timeF         = "time"
	deltaTF       = "deltaT"
	speedF        = "speed"
	lengthF       = "length"
	lambdaF       = "lambda"
	periodNumberF = "periodNumber"

	defaultMode         = "v"
	defaultFormat       = "html"
	defaultSource       = "."
	defaultOutput       = "."
	defaultTime         = "64"
	defaultDeltaT       = "0.000001"
	defaultSpeed        = "0.01"
	defaultLength       = "1.28"
	defaultLambda       = "0.000001550"
	defaultPeriodNumber = "1"

	modeFlagUsage         = "The mode of the output"
	formatFlagUsage       = "The format of the output"
	sourceFlagUsage       = "The direcory with source files or a single file"
	outputFlagUsage       = "The output directory"
	timeFlagUsage         = "The time of the signal"
	deltaTFlagUsage       = "The time step"
	speedFlagUsage        = "The speed of the translator"
	lengthFlagUsage       = "The difference of the path created by the translator"
	lambdaFlagUsage       = "The wavelength of the signal"
	periodNumberFlagUsage = "The number of periods"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-quit
		cancel()
	}()

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	cmd := NewCmd(func(cmd *cobra.Command, _ []string) error {
		mode, err := mode.UnmarshalText(cmd.Flag(modeF).Value.String())
		if err != nil {
			return fmt.Errorf("failed to parse mode: %w", err)
		}
		format, err := format.UnmarshalText(cmd.Flag(formatF).Value.String())
		if err != nil {
			return fmt.Errorf("failed to parse format: %w", err)
		}
		source := cmd.Flag(sourceF).Value.String()
		output := cmd.Flag(outputF).Value.String()
		t, err := strconv.ParseFloat(cmd.Flag(timeF).Value.String(), 64)
		if err != nil {
			return fmt.Errorf("failed to parse time: %w", err)
		}
		deltaT, err := strconv.ParseFloat(cmd.Flag(deltaTF).Value.String(), 64)
		if err != nil {
			return fmt.Errorf("failed to parse deltaT: %w", err)
		}
		speed, err := strconv.ParseFloat(cmd.Flag(speedF).Value.String(), 64)
		if err != nil {
			return fmt.Errorf("failed to parse speed: %w", err)
		}
		length, err := strconv.ParseFloat(cmd.Flag(lengthF).Value.String(), 64)
		if err != nil {
			return fmt.Errorf("failed to parse length: %w", err)
		}
		lambda, err := strconv.ParseFloat(cmd.Flag(lambdaF).Value.String(), 64)
		if err != nil {
			return fmt.Errorf("failed to parse lambda: %w", err)
		}
		perNum, err := strconv.Atoi(cmd.Flag(periodNumberF).Value.String())
		if err != nil {
			return fmt.Errorf("failed to parse period number: %w", err)
		}

		app := app.New(source, output,
			&parameters.Parameters{
				Mode:         mode,
				Format:       format,
				Time:         t,
				DeltaT:       deltaT,
				Speed:        speed,
				Length:       length,
				Lambda:       lambda,
				PeriodNumber: perNum,
			},
		)

		return app.Run(cmd.Context())
	})

	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func NewCmd(run func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interferometer",
		Short: "Interferometer is a tool to generate visibility data",
		Long: `Interferometer is a tool to generate visibility data. 
It reads binary files from the source directory, calculates visibility data and saves it to the output directory.`,
		RunE: run,
	}

	cmd.Flags().String(sourceF, defaultSource, sourceFlagUsage)
	cmd.Flags().String(outputF, defaultOutput, outputFlagUsage)
	cmd.Flags().String(timeF, defaultTime, timeFlagUsage)
	cmd.Flags().String(deltaTF, defaultDeltaT, deltaTFlagUsage)
	cmd.Flags().String(speedF, defaultSpeed, speedFlagUsage)
	cmd.Flags().String(lengthF, defaultLength, lengthFlagUsage)
	cmd.Flags().String(lambdaF, defaultLambda, lambdaFlagUsage)
	cmd.Flags().String(periodNumberF, defaultPeriodNumber, periodNumberFlagUsage)

	return cmd
}
