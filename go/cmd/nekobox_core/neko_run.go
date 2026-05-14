package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
)

// createBox creates and starts a sing-box instance from JSON config
func createBox(configContent []byte) (*box.Box, context.CancelFunc, error) {
	// Create context with protocol registration
	ctx, cancel := context.WithCancel(context.Background())
	ctx = include.Context(ctx)

	// Parse option.Options from JSON (extended context registers protocol options)
	options, err := sjson.UnmarshalExtendedContext[option.Options](ctx, configContent)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("decode config: %w", err)
	}

	// Disable color in log options (for neko.log file output)
	if options.Log == nil {
		options.Log = &option.LogOptions{}
	}
	options.Log.DisableColor = true

	// Create box
	instance, err := box.New(box.Options{
		Context:           ctx,
		Options:           options,
		PlatformLogWriter: &nekoPlatformWriter{},
	})
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("create service: %w", err)
	}

	// Start box
	err = instance.Start()
	if err != nil {
		cancel()
		instance.Close()
		return nil, nil, fmt.Errorf("start service: %w", err)
	}

	return instance, cancel, nil
}

// singboxMain implements a minimal sing-box CLI (run/check commands)
func singboxMain() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nekobox_core <run|check> -c <config>")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "run":
		singboxRun()
	case "check":
		singboxCheck()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func singboxRun() {
	var configPath string
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "-c" && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	if configPath == "" {
		fmt.Println("Usage: nekobox_core run -c <config>")
		os.Exit(1)
	}

	configContent, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Read config error: %v\n", err)
		os.Exit(1)
	}

	instance, cancel, err := createBox(configContent)
	if err != nil {
		fmt.Printf("Create box error: %v\n", err)
		os.Exit(1)
	}
	defer cancel()
	defer instance.Close()

	fmt.Println("sing-box started")

	// Wait for signal
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
	<-osSignals

	fmt.Println("sing-box stopping...")
}

func singboxCheck() {
	var configPath string
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "-c" && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	if configPath == "" {
		fmt.Println("Usage: nekobox_core check -c <config>")
		os.Exit(1)
	}

	configContent, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Read config error: %v\n", err)
		os.Exit(1)
	}

	// Create context with protocol registration
	ctx := include.Context(context.Background())

	// Parse config
	_, err = sjson.UnmarshalExtendedContext[option.Options](ctx, configContent)
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration is valid")
}
