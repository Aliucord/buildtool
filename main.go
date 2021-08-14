package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

type cfg struct {
	Aliucord, Plugins, AndroidSDK, AndroidSDKVersion, Outputs, OutputsPlugins string
}

const (
	RESET   = "\033[0m"
	ERROR   = "\033[1;31m"
	SUCCESS = "\033[1;32m"
	WARN    = "\033[1;33m"
	INFO    = "\033[1;34m"
)

var (
	configPath = flag.String("config", "config.json", "Config path")
	plugin     = flag.String("plugin", "", "Plugin name to build")
	outName    = flag.String("output", "", "Output file name")
	injector = flag.Bool("injector", false, "Build the injector")

	config cfg
)

func main() {
	flag.StringVar(plugin, "p", *plugin, "Alias for plugin")
	flag.StringVar(outName, "o", *outName, "Alias for output")
	flag.Parse()

	log.SetFlags(log.Lshortfile)

	b, err := ioutil.ReadFile(*configPath)
	handleErr(err)
	handleErr(json.Unmarshal(b, &config))

	if config.AndroidSDKVersion == "" {
		colorPrint(WARN, "NOTE: AndroidSDKVersion not set in config. Defaulting to v29. This will change to v30 in the future.")
		config.AndroidSDKVersion = "29" // NOTE: warn in next versions to update config and use android 30 sdk
	}

	err = exec.Command("d8", "--version").Run()
	if err != nil {
		buildToolNotFound("d8")
	}
	err = exec.Command("aapt2", "version").Run()
	if err != nil {
		buildToolNotFound("aapt2")
	}

	if *injector {
		build("Injector")
	} else if *plugin == "" {
		build("Aliucord")
	} else if *plugin == "*" {
		regex := regexp.MustCompile(`':(\w+)'`)
		buffer := bytes.NewBufferString("")

		gradlew(buffer, config.Plugins, "projects")

		plugins := regex.FindAllStringSubmatch(buffer.String(), -1)

		for i, plugin := range plugins {
			pluginName := plugin[1] //Match the first group, since at index 0 we have the full string

			if pluginName == "Aliucord" || pluginName == "DiscordStubs" {
				continue
			}

			if i > 0 {
				fmt.Println()
			}

			colorPrint(INFO, "Building plugin: " + pluginName)
			buildPlugin(pluginName)
		}
	} else {
		buildPlugin(strings.TrimSpace(*plugin))
	}
}

func buildToolNotFound(tool string) {
	fatal(tool + " not found. Please add the Android build-tools (Android/Sdk/build-tools/VERSION) to your PATH and try again")
}
