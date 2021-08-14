package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func build(project string) {
	gradlew(os.Stdout, config.Aliucord, ":" + project + ":compileDebugJavaWithJavac")

	javacBuild, err := filepath.Abs(fmt.Sprintf("%s/%s/build/intermediates/javac/debug", config.Aliucord, project))
	handleErr(err)

	f, _ := os.Create(javacBuild + "classes.zip")
	zipw := zip.NewWriter(f)
	zipAndD8(f, zipw, javacBuild, "/aliucord.zip", config.Outputs)

	out := project + ".dex"
	if *outName != "" {
		out = *outName
		if !strings.HasSuffix(out, ".dex") {
			out += ".dex"
		}
	}

	os.Rename(config.Outputs + "/classes.dex", config.Outputs + "/" + out)

	colorPrint(success, "Successfully built " + project)
}

func buildPlugin(pluginName string) {
	plugin, err := filepath.Abs(config.Plugins + "/" + pluginName)
	handleErr(err)
	_, err = os.Stat(plugin)
	handleErr(err)

	gradlew(os.Stdout, config.Plugins, pluginName+":compileDebugJavaWithJavac")

	javacBuild := plugin + "/build/intermediates/javac/debug"

	f, _ := os.Create(javacBuild + "classes.zip")
	zipw := zip.NewWriter(f)
	zipAndD8(f, zipw, javacBuild, "/classes.zip", config.OutputsPlugins)

	outputsPlugins, err := filepath.Abs(config.OutputsPlugins)
	handleErr(err)

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
			execCmd(os.Stdout, outputsPlugins, "aapt2", "link", "-I", config.AndroidSDK + "/platforms/android-" + config.AndroidSDKVersion + "/android.jar",
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
	colorPrint(success, "Successfully built plugin " + pluginName)
}

func zipAndD8(f* os.File, zipw* zip.Writer, javacBuild, zipName, outputPath string) {
	filepath.Walk(javacBuild + "/classes", func(path string, f os.FileInfo, err error) error {
		if err != nil {
			colorPrint(red, err)
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

	output, err := filepath.Abs(outputPath)
	handleErr(err)

	execCmd(os.Stdout, output, "d8", javacBuild + zipName)
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
