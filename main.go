package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

type cfg struct {
	Aliucord, Plugins, AndroidSDK, AndroidSDKVersion, Outputs, OutputsPlugins string
}

const (
	info    = "\033[1;34m%s\033[0m"
	success = "\033[1;32m%s\033[0m"
)

var (
	configPath = flag.String("config", "config.json", "Config path")
	plugin     = flag.String("plugin", "", "Plugin name to build")
	outName    = flag.String("output", "", "Output file name")

	config cfg
)

func main() {
	flag.StringVar(plugin, "p", *plugin, "Alias for plugin")
	flag.StringVar(outName, "o", *outName, "Alias for output")
	flag.Parse()

	log.SetFlags(log.Lshortfile)

	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Fatal(err)
	}

	if config.AndroidSDKVersion == "" {
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

	if *plugin == "" {
		build()
	} else if *plugin == "*" {
		b, err = ioutil.ReadFile(config.Plugins + "/settings.gradle")
		file := strings.Split(string(b), "\n")

		for i, ln := range file {
			if len(strings.TrimSpace(ln)) == 0 {
				continue
			}

			if strings.Contains(ln, "rootProject.name") {
				break
			}

			if i > 0 {
				fmt.Print("\n")
			}

			pluginName := strings.TrimSpace(strings.Replace(strings.ReplaceAll(strings.ReplaceAll(ln, `"`, ""), "'", ""), "include :", "", 1))
			fmt.Printf(info+"\n", "Building plugin: "+pluginName)
			buildPlugin(pluginName)
		}
	} else {
		buildPlugin(strings.TrimSpace(*plugin))
	}
}

func buildToolNotFound(tool string) {
	log.Fatal(tool + " not found. Please add the Android build-tools (Android/Sdk/build-tools/VERSION) to your PATH and try again")
}
