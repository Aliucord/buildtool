package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func build() {
	gradlew(os.Stdout, config.Aliucord, ":Aliucord:compileDebugJavaWithJavac")

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

	execCmd(os.Stdout, config.Outputs, "d8", javacBuild+"/aliucord.zip")

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

	gradlew(os.Stdout, config.Plugins, pluginName+":compileDebugJavaWithJavac")

	javacBuild := plugin + "/build/intermediates/javac/debug"
	f, _ := os.Create(javacBuild + "/classes.zip")
	zipw := zip.NewWriter(f)

	filepath.Walk(javacBuild+"/classes", func(path string, f os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)
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
	execCmd(os.Stdout, outputsPlugins, "d8", javacBuild+"/classes.zip")

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

			execCmd(os.Stdout, outputsPlugins, "aapt2", "compile", "--dir", src+"/res", "-o", "tmpres.zip")
			execCmd(os.Stdout, outputsPlugins, "aapt2", "link", "-I", config.AndroidSDK+"/platforms/android-"+config.AndroidSDKVersion+"/android.jar",
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

			writePluginEntry(zipw, pluginName)

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

	writePluginEntry(zipw, pluginName)
}
