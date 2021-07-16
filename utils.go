package main

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func gradlew(stdout io.Writer, dir string, args ...string) {
	if runtime.GOOS == "windows" {
		execCmd(stdout, dir, "cmd", "/k", "gradlew.bat "+strings.Join(args, " ")+" && exit")
	} else {
		execCmd(stdout, dir, "./gradlew", args...)
	}
}

func execCmd(stdout io.Writer, dir string, c string, args ...string) {
	cmd := exec.Command(c, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func writePluginEntry(zipw *zip.Writer, pluginName string) {
	zipFilew, _ := zipw.Create("ac-plugin")
	zipFilew.Write([]byte(pluginName))
}
