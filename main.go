package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &config)
	if err != nil {
		panic(err)
	}

	if *plugin == "" {
		build()
	} else if *plugin == "*" {
		b, err = ioutil.ReadFile(config.Plugins + "/settings.gradle")
		file := strings.Split(string(b), "\n")

		for i, ln := range file {
			if strings.Contains(ln, "rootProject.name") {
				break
			}

			if i > 0 {
				fmt.Print("\n")
			}

			pluginName := strings.TrimSpace(strings.Replace(strings.ReplaceAll(strings.ReplaceAll(ln, `"`, ""), "'", ""), "include :", "", 1))
			fmt.Printf(info+"\n", "Builiding plugin: "+pluginName)
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
		panic(err)
	}
	f, _ := os.Create(javacBuild + "/aliucord.zip")
	zipw := zip.NewWriter(f)

	filepath.Walk(javacBuild+"/classes/com/aliucord", func(path string, f os.FileInfo, err error) error {
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
	gradlew(config.Plugins, pluginName+":compileDebugJavaWithJavac")

	javacBuild, err := filepath.Abs(config.Plugins + "/" + pluginName + "/build/intermediates/javac/debug")
	if err != nil {
		panic(err)
	}
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

	execCmd(config.OutputsPlugins, "d8", javacBuild+"/classes.zip")

	out := pluginName + ".apk"
	if *outName != "" {
		out = *outName
		if !strings.HasSuffix(out, ".apk") {
			out += ".apk"
		}
	}

	src, err := filepath.Abs(config.Plugins + "/" + pluginName + "/src/main")
	if err == nil {
		files, err := ioutil.ReadDir(src + "/res")
		if err == nil && len(files) > 0 {
			tmpApk := config.OutputsPlugins + "/" + pluginName + "-tmp.apk"

			execCmd(config.OutputsPlugins, "aapt2", "compile", "--dir", src+"/res", "-o", "tmpres.zip")
			execCmd(config.OutputsPlugins, "aapt2", "link", "-I", config.AndroidSDK+"/platforms/android-29/android.jar",
				"-R", "tmpres.zip", "--manifest", src+"/AndroidManifest.xml", "-o", tmpApk)
			os.Remove(config.OutputsPlugins + "/tmpres.zip")

			zipr, _ := zip.OpenReader(tmpApk)
			f, _ = os.Create(config.OutputsPlugins + "/" + out)
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

			f, _ = os.Open(config.OutputsPlugins + "/classes.dex")
			zipFilew, _ := zipw.Create("classes.dex")
			io.Copy(zipFilew, f)
			f.Close()

			os.Remove(tmpApk)
		} else {
			makeZipWithClasses(out)
		}
	} else {
		makeZipWithClasses(out)
	}

	os.Remove(config.OutputsPlugins + "/classes.dex")
	fmt.Printf("\n"+success+"\n", "Successfully built plugin: "+pluginName)
}

func makeZipWithClasses(out string) {
	f, _ := os.Create(config.OutputsPlugins + "/" + out)
	defer f.Close()
	zipw := zip.NewWriter(f)
	defer zipw.Close()

	f, _ = os.Open(config.OutputsPlugins + "/classes.dex")
	zipFilew, _ := zipw.Create("classes.dex")
	io.Copy(zipFilew, f)
	f.Close()
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
		panic(err)
	}
}
