package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type cfg struct {
	Aliucord, Plugins, AndroidSDK, Outputs, OutputsPlugins string
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

	err = exec.Command("d8", "--version").Run()
	if err != nil {
		log.Fatal("d8 not found. Please add the Android build-tools (Android/Sdk/build-tools/VERSION) to your PATH and try again")
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

func build() {
	gradlew(config.Aliucord, ":Aliucord:compileDebugJavaWithJavac")

	javacBuild, err := filepath.Abs(config.Aliucord + "/Aliucord/build/intermediates/javac/debug")
	if err != nil {
		log.Fatal(err)
	}
	f, _ := os.Create(javacBuild + "/aliucord.zip")
	zipw := zip.NewWriter(f)

	filepath.Walk(javacBuild+"/classes", func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}

		file, _ := os.Open(path)
		defer file.Close()

		zipf, _ := zipw.Create(strings.Split(strings.ReplaceAll(path, "\\", "/"), "javac/debug/classes/")[1])
		io.Copy(zipf, file)

		return nil
	})

	zipw.Close()
	f.Close()

	execCmd(config.Outputs, "d8", javacBuild+"/aliucord.zip")

	out := "Aliucord.dex"
	if *outName != "" {
		out = *outName
		if !strings.HasSuffix(out, ".dex") {
			out += ".dex"
		}
	}
	os.Rename(config.Outputs+"/classes.dex", config.Outputs+"/"+out)

	fmt.Printf("\n"+success+"\n", "Successfully built Aliucord")
}

func buildPlugin(pluginName string) {
	plugin, err := filepath.Abs(config.Plugins + "/" + pluginName)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = os.Stat(plugin); err != nil {
		log.Fatal(err)
	}

	gradlew(config.Plugins, pluginName+":compileDebugJavaWithJavac")

	javacBuild := plugin + "/build/intermediates/javac/debug"
	f, _ := os.Create(javacBuild + "/classes.zip")
	zipw := zip.NewWriter(f)

	filepath.Walk(javacBuild+"/classes", func(path string, f os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if f.IsDir() {
			return nil
		}

		file, _ := os.Open(path)
		defer file.Close()

		zipf, _ := zipw.Create(strings.Split(strings.ReplaceAll(path, "\\", "/"), "javac/debug/classes/")[1])
		io.Copy(zipf, file)

		return nil
	})

	zipw.Close()
	f.Close()

	outputsPlugins, err := filepath.Abs(config.OutputsPlugins)
	if err != nil {
		log.Fatal(err)
	}
	execCmd(outputsPlugins, "d8", javacBuild+"/classes.zip")

	out := pluginName + ".zip"
	if *outName != "" {
		out = *outName
		if !strings.HasSuffix(out, ".zip") {
			out += ".zip"
		}
	}

	src, err := filepath.Abs(config.Plugins + "/" + pluginName + "/src/main")
	if err == nil {
		files, err := ioutil.ReadDir(src + "/res")
		if err == nil && len(files) > 0 {
			tmpApk := outputsPlugins + "/" + pluginName + "-tmp.apk"

			execCmd(outputsPlugins, "aapt2", "compile", "--dir", src+"/res", "-o", "tmpres.zip")
			execCmd(outputsPlugins, "aapt2", "link", "-I", config.AndroidSDK+"/platforms/android-29/android.jar",
				"-R", "tmpres.zip", "--manifest", src+"/AndroidManifest.xml", "-o", tmpApk)
			os.Remove(outputsPlugins + "/tmpres.zip")

			zipr, _ := zip.OpenReader(tmpApk)
			f, _ = os.Create(outputsPlugins + "/" + out)
			defer f.Close()
			zipw = zip.NewWriter(f)
			defer zipw.Close()

			for _, zipFile := range zipr.File {
				if zipFile.Name == "AndroidManifest.xml" {
					continue
				}

				zipFiler, _ := zipFile.Open()
				zipFilew, _ := zipw.Create(zipFile.Name)
				io.Copy(zipFilew, zipFiler)
				zipFiler.Close()
			}
			zipr.Close()

			f, _ = os.Open(outputsPlugins + "/classes.dex")
			zipFilew, _ := zipw.Create("classes.dex")
			io.Copy(zipFilew, f)
			f.Close()

			zipFilew, _ = zipw.Create("ac-plugin")
			zipFilew.Write([]byte(pluginName))

			os.Remove(tmpApk)
		} else {
			makeZipWithClasses(out, pluginName)
		}
	} else {
		makeZipWithClasses(out, pluginName)
	}

	os.Remove(outputsPlugins + "/classes.dex")
	fmt.Printf("\n"+success+"\n", "Successfully built plugin: "+pluginName)
}

func makeZipWithClasses(out, pluginName string) {
	f, _ := os.Create(config.OutputsPlugins + "/" + out)
	defer f.Close()
	zipw := zip.NewWriter(f)
	defer zipw.Close()

	f, _ = os.Open(config.OutputsPlugins + "/classes.dex")
	zipFilew, _ := zipw.Create("classes.dex")
	io.Copy(zipFilew, f)
	f.Close()

	zipFilew, _ = zipw.Create("ac-plugin")
	zipFilew.Write([]byte(pluginName))
}

func gradlew(dir string, args ...string) {
	if runtime.GOOS == "windows" {
		execCmd(dir, "cmd", "/k", "gradlew.bat "+strings.Join(args, " ")+" && exit")
	} else {
		execCmd(dir, "./gradlew", args...)
	}
}

func execCmd(dir, c string, args ...string) {
	cmd := exec.Command(c, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
