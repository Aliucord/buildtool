package main

import (
	"archive/zip"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

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

func writePluginEntry(zipw *zip.Writer, pluginName string) {
	zipFilew, _ := zipw.Create("ac-plugin")
	zipFilew.Write([]byte(pluginName))
}
